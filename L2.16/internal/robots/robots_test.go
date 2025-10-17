package robots_test

import (
	"goWget/internal/robots"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestRobotsFetchAndAllowed(t *testing.T) {
	robotsTxt := `
		# Comment line
		User-agent: *
		Disallow: /private
		Disallow: /tmp

		User-agent: OtherBot
		Disallow: /other
		`
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/robots.txt" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(robotsTxt))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer ts.Close()

	u, _ := url.Parse(ts.URL)
	r := robots.NewRobots(http.DefaultClient, u)

	if err := r.Fetch(); err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	tests := []struct {
		path    string
		allowed bool
	}{
		{"/", true},
		{"/public", true},
		{"/private/data", false},
		{"/tmp/file.txt", false},
		{"/other/page", true},
	}

	for _, tt := range tests {
		u.Path = tt.path
		got := r.Allowed(u)
		if got != tt.allowed {
			t.Errorf("Allowed(%s) = %v, want %v", tt.path, got, tt.allowed)
		}
	}
}

func TestRobotsFetch404(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	u, _ := url.Parse(ts.URL)
	r := robots.NewRobots(http.DefaultClient, u)

	err := r.Fetch()
	if err == nil {
		t.Fatal("Expected error for 404 robots.txt")
	}
}
