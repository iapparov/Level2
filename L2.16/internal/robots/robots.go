package robots

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

type Robots struct {
	client   *http.Client
	root     *url.URL
	disallow []string // префиксы, которые запрещены
}

func NewRobots(client *http.Client, root *url.URL) *Robots {
	return &Robots{client: client, root: root}
}

func (r *Robots) Fetch() error {
	robotsURL := *r.root // copy
	robotsURL.Path = "/robots.txt"
	resp, err := r.client.Get(robotsURL.String())
	if err != nil {
		return err
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			fmt.Fprintf(os.Stderr, "error closing response body: %v\n", cerr)
		}
	}()
	if resp.StatusCode != 200 {
		return fmt.Errorf("robots.txt: HTTP %d", resp.StatusCode)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	r.parse(string(b))
	return nil
}

func (r *Robots) parse(body string) {
	lines := strings.Split(body, "\n")
	uaMatched := false
	var localDisallow []string
	// Простая логика: если встречаем "User-agent: *" — начинаем собирать Disallow
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		k := strings.ToLower(strings.TrimSpace(parts[0]))
		v := strings.TrimSpace(parts[1])
		if k == "user-agent" {
			if v == "*" {
				uaMatched = true
			} else {
				uaMatched = false
			}
			continue
		}
		if uaMatched && k == "disallow" {
			if v == "" {
				// пустой Disallow — значит разрешено всё
				continue
			}
			localDisallow = append(localDisallow, v)
		}
	}
	r.disallow = localDisallow
}

// Allowed проверяет, разрешён ли URL для скачивания
func (r *Robots) Allowed(u *url.URL) bool {
	if r == nil || len(r.disallow) == 0 {
		return true
	}
	// сравниваем путь с префиксами disallow
	for _, p := range r.disallow {
		if strings.HasPrefix(u.Path, p) {
			return false
		}
	}
	return true
}
