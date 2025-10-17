package downloader

import (
	"crypto/sha1"
	"fmt"
	"goWget/internal/robots"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type Downloader struct {
	root   *url.URL
	client *http.Client
	outDir string
	robots *robots.Robots
}

func NewDownloader(root string, outDir string, timeout time.Duration) *Downloader {
	u, _ := url.Parse(root)
	if outDir == "" {
		host := strings.ReplaceAll(u.Host, ":", "_")
		outDir = fmt.Sprintf("./%s", host)
	}
	d := &Downloader{
		root:   u,
		client: &http.Client{Timeout: timeout},
		outDir: outDir,
	}
	// Попробуем загрузить robots.txt (если есть)
	d.robots = robots.NewRobots(d.client, d.root)
	if err := d.robots.Fetch(); err != nil {
		// не фатально — просто лог
		fmt.Fprintf(os.Stderr, "robots.txt: %v\n", err)
	}
	return d
}

// fetch делает HTTP GET и возвращает тело + content-type
func (d *Downloader) Fetch(u *url.URL) ([]byte, string, error) {
	if !d.robots.Allowed(u) {
		return nil, "", fmt.Errorf("blocked by robots.txt: %s", u.String())
	}
	resp, err := d.client.Get(u.String())
	if err != nil {
		return nil, "", err
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			fmt.Fprintf(os.Stderr, "error closing response body: %v\n", cerr)
		}
	}()
	if resp.StatusCode >= 400 {
		return nil, "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}
	return b, resp.Header.Get("Content-Type"), nil
}

// localPathFor — формируем локальный путь по URL
func (d *Downloader) LocalPathFor(u *url.URL, contentType string) string {
	p := u.Path
	if p == "" || strings.HasSuffix(p, "/") {
		p = path.Join(p, "index.html")
	}
	ext := path.Ext(p)
	if ext == "" {
		if strings.Contains(contentType, "text/html") {
			p = path.Join(p, "index.html")
		} else {
			h := sha1.Sum([]byte(u.String()))
			p = path.Join(filepath.Dir(p), fmt.Sprintf("resource-%x", h[:6]))
		}
	}
	if u.RawQuery != "" {
		q := d.sanitizeFilename(u.RawQuery)
		p = p + "_" + q
	}
	local := filepath.Join(d.outDir, filepath.FromSlash(path.Clean(p)))
	return local
}

func (d *Downloader) sanitizeFilename(s string) string {
	var unsafeRe = regexp.MustCompile(`[^a-zA-Z0-9._-]`)
	if s == "" {
		return "q"
	}
	res := unsafeRe.ReplaceAllString(s, "_")
	if len(res) > 100 {
		res = res[:100]
	}
	return res
}

func (d *Downloader) SaveToFile(localPath string, data []byte) error {
	down := filepath.Dir(localPath)
	if err := os.MkdirAll(down, 0755); err != nil {
		return err
	}
	return os.WriteFile(localPath, data, 0644)
}
