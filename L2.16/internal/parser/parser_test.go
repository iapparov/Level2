package parser

import (
	"bytes"
	"goWget/internal/downloader"
	"golang.org/x/net/html"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

func TestWalkSimpleHTML(t *testing.T) {
	htmlStr := `<html>
	<body>
	<a href="/page1.html">Page</a>
	<img src="/img/pic.png">
	<link rel="stylesheet" href="/css/style.css">
	</body></html>`

	base, _ := url.Parse("https://example.com/")
	doc, _ := html.Parse(strings.NewReader(htmlStr))
	c := &Crawler{root: base}

	assets, pages := c.walk(doc, base)

	var assetList, pageList []string
	for _, a := range assets {
		assetList = append(assetList, a.Path)
	}
	for _, p := range pages {
		pageList = append(pageList, p.Path)
	}

	wantAssets := []string{"/page1.html", "/img/pic.png", "/css/style.css"}
	wantPages := []string{"/page1.html"}

	for _, w := range wantAssets {
		if !contains(assetList, w) {
			t.Errorf("expected asset %s not found in %v", w, assetList)
		}
	}
	for _, w := range wantPages {
		if !contains(pageList, w) {
			t.Errorf("expected page %s not found in %v", w, pageList)
		}
	}
}

func contains(list []string, val string) bool {
	for _, s := range list {
		if s == val {
			return true
		}
	}
	return false
}

func TestRewriteUpdatesAttributes(t *testing.T) {
	htmlStr := `<html><body>
	<a href="/page1.html">Page</a>
	<img src="/img/pic.png">
	</body></html>`

	doc, _ := html.Parse(strings.NewReader(htmlStr))
	base, _ := url.Parse("https://example.com/")

	localMap := map[string]string{
		"https://example.com/page1.html":  "./mirror/page1.html",
		"https://example.com/img/pic.png": "./mirror/img_pic.png",
	}
	root := "https://example.com"
	tmpDir := t.TempDir()

	down := downloader.NewDownloader(root, tmpDir, 0)
	c := &Crawler{root: base}
	c.rewrite(doc, base, localMap, down)

	var buf bytes.Buffer
	_ = html.Render(&buf, doc)
	out := buf.String()

	if !strings.Contains(out, "href=\"mirror/page1.html\"") && !strings.Contains(out, "href=\"./mirror/page1.html\"") {
		t.Errorf("rewrite did not update link href: %s", out)
	}
	if !strings.Contains(out, "src=\"mirror/img_pic.png\"") && !strings.Contains(out, "src=\"./mirror/img_pic.png\"") {
		t.Errorf("rewrite did not update img src: %s", out)
	}
}

func TestExtractAndProcessHTML(t *testing.T) {
	htmlStr := `<html><body>
	<a href="/sub.html">Sub</a>
	<img src="/pic.png">
	</body></html>`

	base, _ := url.Parse("https://example.com/")

	root := "https://example.com"
	tmpDir := t.TempDir()
	down := downloader.NewDownloader(root, tmpDir, 0)

	c := &Crawler{
		root:     base,
		maxDepth: 1,
		sem:      make(chan struct{}, 2),
		visited:  make(map[string]struct{}),
	}

	result, err := c.extractAndProcessHTML(base, []byte(htmlStr), 1, down)
	if err != nil {
		t.Fatalf("extractAndProcessHTML error: %v", err)
	}

	out := string(result)
	if !strings.Contains(out, "<img") || !strings.Contains(out, "<a") {
		t.Errorf("output HTML malformed: %s", out)
	}
}

func TestMarkVisited(t *testing.T) {
	c := &Crawler{visited: make(map[string]struct{})}
	url := "https://example.com/test"
	if !c.markVisited(url) {
		t.Errorf("first call should return true")
	}
	if c.markVisited(url) {
		t.Errorf("second call should return false")
	}
}

func TestStartSavesRootAssetsAndSubpage(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = io.WriteString(w, `<html><head></head><body>
                <a href="/sub.html">sub</a>
                <img src="/img.png" />
            </body></html>`)
		case "/sub.html":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = io.WriteString(w, `<html><body><p>SUBPAGE</p></body></html>`)
		case "/img.png":
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write([]byte{0x89, 0x50, 0x4E, 0x47})
		case "/robots.txt":
			http.NotFound(w, r)
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	out := t.TempDir()

	down := downloader.NewDownloader(ts.URL, out, 5*time.Second)

	c, err := NewCrawler(ts.URL, 1, 4)
	if err != nil {
		t.Fatalf("NewCrawler: %v", err)
	}

	if err := c.Start(down); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	startURL, _ := url.Parse(ts.URL)
	startLocal := down.LocalPathFor(startURL, "text/html; charset=utf-8")
	if _, err := os.Stat(startLocal); err != nil {
		t.Fatalf("root file not saved at %s: %v", startLocal, err)
	}
	b, _ := os.ReadFile(startLocal)
	outHTML := string(b)

	if !strings.Contains(outHTML, "sub.html") {
		t.Fatalf("expected root HTML to reference sub.html; got: %s", outHTML)
	}
	if !strings.Contains(outHTML, "img.png") && !strings.Contains(outHTML, "img") {
		t.Fatalf("expected root HTML to reference image; got: %s", outHTML)
	}

	subURL, _ := url.Parse(ts.URL + "/sub.html")
	subLocal := down.LocalPathFor(subURL, "text/html; charset=utf-8")
	if _, err := os.Stat(subLocal); err != nil {
		t.Fatalf("subpage not saved at %s: %v", subLocal, err)
	}
	sb, _ := os.ReadFile(subLocal)
	if !strings.Contains(string(sb), "SUBPAGE") {
		t.Fatalf("subpage content mismatch: %s", string(sb))
	}

	imgURL, _ := url.Parse(ts.URL + "/img.png")
	imgLocal := down.LocalPathFor(imgURL, "image/png")
	if _, err := os.Stat(imgLocal); err != nil {
		t.Fatalf("image not saved at %s: %v", imgLocal, err)
	}
}

func TestStartDepthZeroDoesNotFetchSubpage(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = io.WriteString(w, `<html><head></head><body>
                <a href="/sub.html">sub</a>
            </body></html>`)
		case "/sub.html":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = io.WriteString(w, `<html><body><p>SUBPAGE</p></body></html>`)
		case "/robots.txt":
			http.NotFound(w, r)
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	out := t.TempDir()
	down := downloader.NewDownloader(ts.URL, out, 5*time.Second)

	c, err := NewCrawler(ts.URL, 0, 2)
	if err != nil {
		t.Fatalf("NewCrawler: %v", err)
	}
	if err := c.Start(down); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	startURL, _ := url.Parse(ts.URL)
	startLocal := down.LocalPathFor(startURL, "text/html; charset=utf-8")
	if _, err := os.Stat(startLocal); err != nil {
		t.Fatalf("root file not saved at %s: %v", startLocal, err)
	}

	subLocal := down.LocalPathFor(&url.URL{Scheme: "http", Host: startURL.Host, Path: "/sub.html"}, "text/html; charset=utf-8")

	if _, err := os.Stat(subLocal); err == nil {

		t.Fatalf("expected subpage NOT to be saved at %s (depth=0), but it exists", subLocal)
	} else {
		if !os.IsNotExist(err) {
			t.Fatalf("unexpected error checking subpage: %v", err)
		}
	}
}

func TestStartNonHTMLStartSavesResource(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write([]byte{1, 2, 3})
			return
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()

	out := t.TempDir()
	down := downloader.NewDownloader(ts.URL, out, 5*time.Second)

	c, err := NewCrawler(ts.URL, 1, 1)
	if err != nil {
		t.Fatalf("NewCrawler: %v", err)
	}
	if err := c.Start(down); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	startURL, _ := url.Parse(ts.URL)
	local := down.LocalPathFor(startURL, "image/png")
	if _, err := os.Stat(local); err != nil {
		t.Fatalf("expected resource saved at %s: %v", local, err)
	}

	b, err := os.ReadFile(local)
	if err != nil {
		t.Fatalf("read saved resource: %v", err)
	}
	if len(b) != 3 {
		t.Fatalf("unexpected saved resource size: %d", len(b))
	}
}
