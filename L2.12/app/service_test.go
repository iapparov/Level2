package app

import (
	"bytes"
	"io"
	"os"
	"reflect"
	"testing"
)

func TestFindMatches(t *testing.T) {
	flags := &FlagOpions{ignore: true}
	lines := []string{"Hello World", "hello there", "Goodbye World"}
	pattern := "hello"
	grep := NewGrep(flags, pattern, lines)

	matchedIdxs := grep.findMatches()
	if len(matchedIdxs) != 2 {
		t.Errorf("Expected 2 matches, got %d", len(matchedIdxs))
	}
}

func TestFindMatchesNoMatch(t *testing.T) {
	flags := &FlagOpions{}
	lines := []string{"Hello World", "hello there", "Goodbye World"}
	pattern := "notfound"
	grep := NewGrep(flags, pattern, lines)

	matchedIdxs := grep.findMatches()
	if len(matchedIdxs) != 0 {
		t.Errorf("Expected 0 matches, got %d", len(matchedIdxs))
	}
}

func TestFindMatchesCountWithIgnore(t *testing.T) {
	flags := &FlagOpions{count: true, ignore: true}
	lines := []string{"Hello World", "hello there", "Goodbye World"}
	pattern := "hello"
	grep := NewGrep(flags, pattern, lines)

	matchedIdxs := grep.findMatches()
	if len(matchedIdxs) != 2 {
		t.Errorf("Expected 2 matches, got %d", len(matchedIdxs))
	}
}

func TestFindVersedMatches(t *testing.T) {
	flags := &FlagOpions{verse: true}
	lines := []string{"hello World", "hello there", "Goodbye World"}
	pattern := "hello"
	grep := NewGrep(flags, pattern, lines)

	matchedIdxs := grep.findMatches()
	if len(matchedIdxs) != 1 {
		t.Errorf("Expected 1 match (versed), got %d", len(matchedIdxs))
	}
}

func TestFindMatchesFixed(t *testing.T) {
	flags := &FlagOpions{fix: true}
	lines := []string{"foo.bar", "foobar", "fooXbar"}
	pattern := "foo.bar"
	grep := NewGrep(flags, pattern, lines)

	matchedIdxs := grep.findMatches()
	if len(matchedIdxs) != 1 {
		t.Errorf("Expected 1 regex match, got %d", len(matchedIdxs))
	}
}

func TestFindMatchesRegexDot(t *testing.T) {
	flags := &FlagOpions{} // без fix
	lines := []string{"foo.bar", "foobar", "fooXbar"}
	pattern := "foo.bar"
	grep := NewGrep(flags, pattern, lines)

	matchedIdxs := grep.findMatches()
	if len(matchedIdxs) != 2 {
		t.Errorf("Expected 2 regex matches, got %d", len(matchedIdxs))
	}
}

func TestFindMatchesIgnoreCase(t *testing.T) {
	flags := &FlagOpions{ignore: true}
	lines := []string{"Hello World", "hello there", "Goodbye World"}
	pattern := "HELLO"
	grep := NewGrep(flags, pattern, lines)

	matchedIdxs := grep.findMatches()
	if len(matchedIdxs) != 2 {
		t.Errorf("Expected 2 matches (ignore case), got %d", len(matchedIdxs))
	}
}

func captureStdout(f func()) (string, error) {
	r, w, err := os.Pipe()
	if err != nil {
		return "", err
	}
	old := os.Stdout
	os.Stdout = w

	// Выполнить функцию, которая пишет в stdout
	f()

	// Восстановить stdout и прочитать результат
	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		return "", err
	}
	_ = r.Close()
	return buf.String(), nil
}

func TestPrintResultSimple(t *testing.T) {
	lines := []string{"Wb", "Hello", "World"}
	flags := &FlagOpions{}
	pattern := "Hello"
	g := NewGrep(flags, pattern, lines)

	out, err := captureStdout(func() {
		g.Filter()
	})

	if err != nil {
		t.Fatalf("Capture error: %v", err)
	}
	if out != "Hello\n" {
		t.Fatalf("Expected: Hello, got: %q", out)
	}
}

func TestPrintResultNumberAndContext(t *testing.T) {
	lines := []string{"Wb", "Hello", "World"}
	flags := &FlagOpions{after: 1, before: 1, number: true}

	g := NewGrep(flags, "Hello", lines)
	out, err := captureStdout(func() {
		g.Filter()
	})
	if err != nil {
		t.Fatalf("Capture error: %v", err)
	}
	want := "1:Wb\n2:Hello\n3:World\n"
	if out != want {
		t.Fatalf("Expected: %q, got: %q", want, out)
	}
}

func TestPrintResultBadContext(t *testing.T) {
	lines := []string{"Wb", "Hello", "World"}
	flags := &FlagOpions{after: 10, before: 10, number: true}

	g := NewGrep(flags, "Hello", lines)
	out, err := captureStdout(func() {
		g.Filter()
	})
	if err != nil {
		t.Fatalf("Capture error: %v", err)
	}
	want := "1:Wb\n2:Hello\n3:World\n"
	if out != want {
		t.Fatalf("Expected: %q, got: %q", want, out)
	}
}

func TestPrintResultCircleBadContext(t *testing.T) {
	lines := []string{"Wb", "Hello", "World"}
	flags := &FlagOpions{circle: 10, number: true}

	g := NewGrep(flags, "Hello", lines)
	out, err := captureStdout(func() {
		g.Filter()
	})
	if err != nil {
		t.Fatalf("Capture error: %v", err)
	}
	want := "1:Wb\n2:Hello\n3:World\n"
	if out != want {
		t.Fatalf("Expected: %q, got: %q", want, out)
	}
}

func TestExpandShortFlags(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want []string
	}{
		{
			name: "simple short flags",
			args: []string{"-F", "-i", "-v"},
			want: []string{"-F", "-i", "-v"},
		},
		{
			name: "combined short flags",
			args: []string{"-vFn"},
			want: []string{"-v", "-F", "-n"},
		},
		{
			name: "short flag with value in same token",
			args: []string{"-A2"},
			want: []string{"-A", "2"},
		},
		{
			name: "short flag with value in next token",
			args: []string{"-B", "2"},
			want: []string{"-B", "2"},
		},
		{
			name: "mixed short flags",
			args: []string{"-Fi", "-A2", "-n"},
			want: []string{"-F", "-i", "-A", "2", "-n"},
		},
	}

	equalSlices := func(a, b []string) bool {
		if len(a) != len(b) {
			return false
		}
		for i := range a {
			if a[i] != b[i] {
				return false
			}
		}
		return true
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ExpandArgs(tt.args); !equalSlices(got, tt.want) {
				t.Errorf("expandArgs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseFlags(t *testing.T) {
	flags := ParseFlags()
	if flags == nil {
		t.Errorf("ParseFlags() error = %v, wantErr %v", flags, "not nil")
	}
}

func TestInputFromStdin(t *testing.T) {
	data := "line1\nline2\n"
	oldStdin := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe error: %v", err)
	}
	_, _ = w.WriteString(data)
	_ = w.Close()

	os.Stdin = r
	t.Cleanup(func() { os.Stdin = oldStdin })

	lines, err := Input([]string{})
	if err != nil {
		t.Fatalf("Input() error = %v", err)
	}
	want := []string{"line1", "line2"}
	if !reflect.DeepEqual(lines, want) {
		t.Fatalf("Input() = %v, want %v", lines, want)
	}
}
