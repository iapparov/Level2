package parser

import (
	"fmt"
	"goWget/internal/downloader"
	"golang.org/x/net/html"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Crawler хранит конфигурацию и состояние
type Crawler struct {
	root      *url.URL
	maxDepth  int
	visited   map[string]struct{}
	visitLock sync.Mutex
	wg        sync.WaitGroup
	sem       chan struct{}
}

func NewCrawler(root string, maxDepth, concurrency int) (*Crawler, error) {
	u, err := url.Parse(root)
	if err != nil {
		return nil, err
	}
	c := &Crawler{
		root:     u,
		maxDepth: maxDepth,
		visited:  make(map[string]struct{}),
		sem:      make(chan struct{}, concurrency),
	}
	return c, nil
}

func (c *Crawler) enqueue() { c.sem <- struct{}{} }
func (c *Crawler) dequeue() { <-c.sem }

// markVisited возвращает true если раньше не было
func (c *Crawler) markVisited(s string) bool {
	c.visitLock.Lock()
	defer c.visitLock.Unlock()
	if _, ok := c.visited[s]; ok {
		return false
	}
	c.visited[s] = struct{}{}
	return true
}

func (c *Crawler) extractAndProcessHTML(base *url.URL, htmlData []byte, depth int, down *downloader.Downloader) ([]byte, error) {
	if depth <= 0 {
		return htmlData, nil
	}

	doc, err := html.Parse(strings.NewReader(string(htmlData)))
	if err != nil {
		return nil, err
	}

	// проход по DOM и сбор ссылок/ресурсов
	assets, pageLinks := c.walk(doc, base)

	// сохраним mapping url localPath для последующей перестановки ссылок
	localMap := make(map[string]string)
	var mu sync.Mutex

	for _, u := range assets {
		s := u.String()
		if u.Hostname() != c.root.Hostname() {
			continue
		}
		if !c.markVisited(s) {
			// уже загружено запомним локальный путь, если он был
			continue
		}
		// планируем загрузку
		c.wg.Add(1)
		go func(u *url.URL) {
			defer c.wg.Done()
			c.enqueue()
			defer c.dequeue()
			data, ct, err := down.Fetch(u)
			if err != nil {
				fmt.Fprintf(os.Stderr, "fetch asset error: %s -> %v\n", u.String(), err)
				return
			}
			local := down.LocalPathFor(u, ct)
			if err := down.SaveToFile(local, data); err != nil {
				fmt.Fprintf(os.Stderr, "save asset error: %s -> %v\n", local, err)
				return
			}
			mu.Lock()
			localMap[u.String()] = local
			mu.Unlock()
		}(u)
	}

	// обработка ссылок на страницы - очереди на рекурсивное скачивание
	var inner sync.WaitGroup
	for _, pl := range pageLinks {
		s := pl.String()
		if pl.Hostname() != c.root.Hostname() {
			continue
		}

		if !c.markVisited(s) {
			continue
		}
		if depth > 0 {
			// ставим в очередь рекурсии
			inner.Add(1)
			go func(pu *url.URL, d int) {
				defer inner.Done()
				c.enqueue()
				defer c.dequeue()
				data, ct, err := down.Fetch(pu)
				if err != nil {
					fmt.Fprintf(os.Stderr, "fetch page error: %s -> %v\n", pu.String(), err)
					return
				}
				local := down.LocalPathFor(pu, ct)
				if strings.Contains(ct, "text/html") {
					processed, err := c.extractAndProcessHTML(pu, data, d-1, down)
					if err != nil {
						fmt.Fprintf(os.Stderr, "process page error: %s -> %v\n", pu.String(), err)

						return
					}
					if err := down.SaveToFile(local, processed); err != nil {
						fmt.Fprintf(os.Stderr, "save page error: %s -> %v\n", local, err)
					}
				} else {
					if err := down.SaveToFile(local, data); err != nil {
						fmt.Fprintf(os.Stderr, "save resource error: %s -> %v\n", local, err)
					}
				}
			}(pl, depth)
		}
	}

	// ждём завершения скачивания ассетов и рекурсивных страниц, запущенных выше
	inner.Wait()

	// после загрузки переписываем ссылки в DOM на относительные локальные пути

	c.rewrite(doc, base, localMap, down)

	// Рендерим DOM в HTML
	var b strings.Builder
	if err := html.Render(&b, doc); err != nil {
		return nil, err
	}
	return []byte(b.String()), nil
}

func (c *Crawler) Start(down *downloader.Downloader) error {
	start := c.root
	c.markVisited(start.String())
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		c.enqueue()
		defer c.dequeue()
		data, ct, err := down.Fetch(start)
		if err != nil {
			fmt.Fprintf(os.Stderr, "fetch start error: %v\n", err)
			return
		}
		local := down.LocalPathFor(start, ct)
		if strings.Contains(ct, "text/html") {
			processed, err := c.extractAndProcessHTML(start, data, c.maxDepth, down)
			if err != nil {
				fmt.Fprintf(os.Stderr, "process start html: %v\n", err)
				return
			}
			if err := down.SaveToFile(local, processed); err != nil {
				fmt.Fprintf(os.Stderr, "save start: %v\n", err)
			}
		} else {
			if err := down.SaveToFile(local, data); err != nil {
				fmt.Fprintf(os.Stderr, "save start resource: %v\n", err)
			}
		}
	}()
	c.wg.Wait()
	return nil
}

func (c *Crawler) rewrite(n *html.Node, base *url.URL, localMap map[string]string, down *downloader.Downloader) {
	if n.Type == html.ElementNode {
		switch n.Data {
		case "a", "img", "script", "link":
			for i, a := range n.Attr {
				if a.Key == "href" || a.Key == "src" {
					resUrl, err := base.Parse(a.Val)
					if err != nil {
						continue
					}
					resUrl.Fragment = ""
					if resUrl.Hostname() != c.root.Hostname() {
						continue
					}
					if local, ok := localMap[resUrl.String()]; ok {
						// Попробуем вычислить относительный путь
						baseLocal := down.LocalPathFor(base, "text/html")
						baseDir := filepath.Dir(baseLocal)

						rel, err := filepath.Rel(baseDir, local)
						if err != nil {
							// fallback: используем прямой путь
							n.Attr[i].Val = filepath.ToSlash(local)
						} else {
							// если относительный путь получился пустым делаем "./"
							if rel == "" {
								rel = "./"
							}
							n.Attr[i].Val = filepath.ToSlash(rel)
						}
					}
				}
			}
		}
	}

	// Рекурсия по дочерним элементам
	for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
		c.rewrite(ch, base, localMap, down)
	}
}

func (c *Crawler) walk(n *html.Node, base *url.URL) (assets []*url.URL, pageLinks []*url.URL) {
	if n.Type == html.ElementNode {
		attrKey := ""
		switch n.Data {
		case "a":
			attrKey = "href"
		case "img", "script":
			attrKey = "src"
		case "link":
			for _, a := range n.Attr {
				if a.Key == "rel" && (a.Val == "stylesheet" || strings.Contains(a.Val, "stylesheet")) {
					attrKey = "href"
					break
				}
			}
		}
		if attrKey != "" {
			for _, a := range n.Attr {
				if a.Key == attrKey {
					resUrl, err := base.Parse(a.Val)
					if err != nil {
						break
					}
					if resUrl.Scheme == "data" {
						break
					}
					resUrl.Fragment = ""
					if resUrl.Hostname() != c.root.Hostname() {
						break
					}
					assets = append(assets, resUrl)
					if n.Data == "a" {
						pageLinks = append(pageLinks, resUrl)
					}
					break
				}
			}
		}
	}
	for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
		as, pls := c.walk(ch, base)
		if len(as) > 0 {
			assets = append(assets, as...)
		}
		if len(pls) > 0 {
			pageLinks = append(pageLinks, pls...)
		}
	}
	return
}
