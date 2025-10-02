package app

import (
	"bytes"
	"io"
	"os"
	"reflect"
	"testing"
)

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
			args: []string{"-f2"},
			want: []string{"-f", "2"},
		},
		{
			name: "short flag with value in next token",
			args: []string{"-B", "2"},
			want: []string{"-B", "2"},
		},
		{
			name: "mixed short flags",
			args: []string{"-Fi", "-d2", "-n"},
			want: []string{"-F", "-i", "-d", "2", "-n"},
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

func TestCutter(t *testing.T) {
	lines := []string{
		"one:two:three",
		"four:five:six",
		"seven:eight:nine",
	}

	flags := &FlagOptions{
		fields:    flagFields{{start: 2, end: -1}},
		delimiter: ":",
		separated: false,
	}
	cut := NewCut(flags, lines)

	out, err := captureStdout(func() {
		cut.Cutter()
	})

	if err != nil {
		t.Fatalf("Capture error: %v", err)
	}
	if out != "two:three\nfive:six\neight:nine\n" {
		t.Fatalf("Expected: %q, got: %q", "two:three\nfive:six\neight:nine\n", out)
	}
}

func TestCutterSeparated(t *testing.T) {
	lines := []string{
		"one:two:three",
		"four-five-six",
		"seven:eight:nine",
	}

	flags := &FlagOptions{
		fields:    flagFields{{start: 1, end: 1}},
		delimiter: ":",
		separated: true,
	}
	cut := NewCut(flags, lines)

	out, err := captureStdout(func() {
		cut.Cutter()
	})

	if err != nil {
		t.Fatalf("Capture error: %v", err)
	}
	if out != "one\nseven\n" {
		t.Fatalf("Expected: %q, got: %q", "one\nseven\n", out)
	}
}

func TestSetSingleField(t *testing.T) {
	var ff flagFields
	err := ff.Set("3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ff) != 1 || ff[0].start != 3 || ff[0].end != 3 {
		t.Errorf("expected [3-3], got %+v", ff)
	}
}

func TestSetRange(t *testing.T) {
	var ff flagFields
	err := ff.Set("2-5")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ff) != 1 || ff[0].start != 2 || ff[0].end != 5 {
		t.Errorf("expected [2-5], got %+v", ff)
	}
}

func TestSetOpenRangeStart(t *testing.T) {
	var ff flagFields
	err := ff.Set("-4")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ff[0].start != 1 || ff[0].end != 4 {
		t.Errorf("expected [1-4], got %+v", ff)
	}
}

func TestSetOpenRangeEnd(t *testing.T) {
	var ff flagFields
	err := ff.Set("3-")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ff[0].start != 3 || ff[0].end != -1 {
		t.Errorf("expected [3--1], got %+v", ff)
	}
}

func TestSetInvalid(t *testing.T) {
	var ff flagFields
	err := ff.Set("abc")
	if err == nil {
		t.Fatal("expected error for invalid input, got nil")
	}
}
