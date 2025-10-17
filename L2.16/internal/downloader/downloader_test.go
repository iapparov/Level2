package downloader

import (
	"crypto/sha1"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSanitizeFilename(t *testing.T) {
	d := NewDownloader("https://example.com", "", time.Second)

	cases := map[string]string{
		"abc123":       "abc123",
		"a/b\\c":       "a_b_c",
		"@#!file?name": "___file_name",
		"":             "q",
	}

	for in, want := range cases {
		got := d.sanitizeFilename(in)
		if got != want {
			t.Errorf("sanitizeFilename(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestLocalPathFor(t *testing.T) {
	d := NewDownloader("https://example.com", "./mirror_example", time.Second)

	u, _ := url.Parse("https://example.com/images/photo.png")
	got := d.LocalPathFor(u, "image/png")
	want := filepath.Join("./mirror_example", "images", "photo.png")
	if got != want {
		t.Errorf("LocalPathFor got %q, want %q", got, want)
	}

	u2, _ := url.Parse("https://example.com/blog/")
	got2 := d.LocalPathFor(u2, "text/html")
	want2 := filepath.Join("./mirror_example", "blog", "index.html")
	if got2 != want2 {
		t.Errorf("LocalPathFor got %q, want %q", got2, want2)
	}

	u3, _ := url.Parse("https://example.com/search?q=cats")
	got3 := d.LocalPathFor(u3, "text/html")
	if !strings.Contains(got3, "search") || !strings.Contains(got3, "_q_cats") {
		t.Errorf("LocalPathFor query not sanitized correctly: %q", got3)
	}

	u4, _ := url.Parse("https://example.com/resource")
	ct := "application/octet-stream"
	got4 := d.LocalPathFor(u4, ct)
	h := sha1.Sum([]byte(u4.String()))
	wantHash := fmt.Sprintf("resource-%x", h[:6])
	if !strings.Contains(got4, wantHash) {
		t.Errorf("LocalPathFor missing hash: %q", got4)
	}
}

func TestSaveToFile(t *testing.T) {
	tmpDir := t.TempDir()
	d := NewDownloader("https://example.com", tmpDir, time.Second)

	path := filepath.Join(tmpDir, "subdir", "file.txt")
	data := []byte("hello world")

	if err := d.SaveToFile(path, data); err != nil {
		t.Fatalf("SaveToFile error: %v", err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}
	if string(b) != "hello world" {
		t.Errorf("expected file content %q, got %q", "hello world", string(b))
	}
}

func TestFetch(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ok":
			w.Header().Set("Content-Type", "text/plain")
			_, _ = io.WriteString(w, "ok")
		case "/err":
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
	defer ts.Close()

	d := NewDownloader(ts.URL, "", time.Second)
	uOK, _ := url.Parse(ts.URL + "/ok")
	data, ct, err := d.Fetch(uOK)
	if err != nil {
		t.Fatalf("Fetch /ok failed: %v", err)
	}
	if string(data) != "ok" || ct != "text/plain" {
		t.Errorf("Fetch /ok got (%q, %q)", string(data), ct)
	}

	uErr, _ := url.Parse(ts.URL + "/err")
	_, _, err = d.Fetch(uErr)
	if err == nil {
		t.Errorf("expected error for /err, got nil")
	}
}
