package app

import (
	"os"
	"reflect"
	"testing"
)

func TestInputFromStdin(t *testing.T) {
	const data = "line1\nline2\n"
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

func TestParseFlags(t *testing.T) {
	flags := ParseFlags()
	if flags == nil {
		t.Errorf("ParseFlags() error = %v, wantErr %v", flags, "not nil")
	}
}

func TestNewSorts(t *testing.T) {
	flags := &FlagOpions{}
	lines := []string{"line1", "line2"}
	sorts := NewSorts(flags, lines)
	if sorts == nil {
		t.Errorf("NewSorts() error = %v, wantErr %v", sorts, "not nil")
	}
}

func TestSort(t *testing.T) {
	flags := &FlagOpions{}
	lines := []string{"line2", "line1"}
	sorts := NewSorts(flags, lines)
	sorts.Sort()
	if sorts.lines[0] != "line1" || sorts.lines[1] != "line2" {
		t.Errorf("Sort() error = %v, wantErr %v", sorts.lines, []string{"line1", "line2"})
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
			args: []string{"-a", "-b", "-c"},
			want: []string{"-a", "-b", "-c"},
		},
		{
			name: "combined short flags",
			args: []string{"-abc"},
			want: []string{"-a", "-b", "-c"},
		},
		{
			name: "short flag with value in same token",
			args: []string{"-k2"},
			want: []string{"-k", "2"},
		},
		{
			name: "short flag with value in next token",
			args: []string{"-k", "2"},
			want: []string{"-k", "2"},
		},
		{
			name: "mixed short flags",
			args: []string{"-ab", "-k2", "-c"},
			want: []string{"-a", "-b", "-k", "2", "-c"},
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

func TestParseHuman(t *testing.T) {
	tests := []struct {
		input string
		want  float64
	}{
		{"", 0},
		{"100", 100},
		{"1K", 1024},
		{"1M", 1024 * 1024},
		{"1G", 1024 * 1024 * 1024},
		{"1T", 1024 * 1024 * 1024 * 1024},
		{"2.5K", 2.5 * 1024},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := parseHuman(tt.input); got != tt.want {
				t.Errorf("parseHuman() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSortMonth(t *testing.T) {
	lines := []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}
	shuffled := []string{"Mar", "Jan", "Dec", "Jul", "Feb", "Nov", "May", "Aug", "Apr", "Jun", "Oct", "Sep"}
	flags := &FlagOpions{month: true}
	sorts := NewSorts(flags, shuffled)
	sorts.Sort()
	for i, line := range sorts.lines {
		if line != lines[i] {
			t.Errorf("SortMonth() failed at index %d: got %s, want %s", i, line, lines[i])
		}
	}
}

func TestSortNumeric(t *testing.T) {
	lines := []string{"10", "2", "33", "25", "5"}
	expected := []string{"2", "5", "10", "25", "33"}
	flags := &FlagOpions{numb: true}
	sorts := NewSorts(flags, lines)
	sorts.Sort()
	for i, line := range sorts.lines {
		if line != expected[i] {
			t.Errorf("SortNumeric() failed at index %d: got %s, want %s", i, line, expected[i])
		}
	}
}
func TestSortHuman(t *testing.T) {
	lines := []string{"1K", "500", "2M", "1G", "750K"}
	expected := []string{"500", "1K", "750K", "2M", "1G"}
	flags := &FlagOpions{human: true}
	sorts := NewSorts(flags, lines)
	sorts.Sort()
	for i, line := range sorts.lines {
		if line != expected[i] {
			t.Errorf("SortHuman() failed at index %d: got %s, want %s", i, line, expected[i])
		}
	}
}

func TestSortColumn(t *testing.T) {
	lines := []string{"b\t2", "a\t3", "c\t1"}
	expected := []string{"a\t3", "b\t2", "c\t1"}
	flags := &FlagOpions{column: 1}
	sorts := NewSorts(flags, lines)
	sorts.Sort()
	for i, line := range sorts.lines {
		if line != expected[i] {
			t.Errorf("SortColumn() failed at index %d: got %s, want %s", i, line, expected[i])
		}
	}
}

func TestSortReverse(t *testing.T) {
	lines := []string{"apple", "banana", "cherry"}
	expected := []string{"cherry", "banana", "apple"}
	flags := &FlagOpions{reverse: true}
	sorts := NewSorts(flags, lines)
	sorts.Sort()
	for i, line := range sorts.lines {
		if line != expected[i] {
			t.Errorf("SortReverse() failed at index %d: got %s, want %s", i, line, expected[i])
		}
	}
}

func TestSortBlanks(t *testing.T) {
	lines := []string{"apple   ", "banana\t\t", "cherry \t "}
	expected := []string{"apple", "banana", "cherry"}
	flags := &FlagOpions{blanks: true}
	sorts := NewSorts(flags, lines)
	sorts.Sort()
	for i, line := range sorts.lines {
		if line != expected[i] {
			t.Errorf("SortBlanks() failed at index %d: got %s, want %s", i, line, expected[i])
		}
	}
}

func TestUnique(t *testing.T) {
	lines := []string{"apple", "banana", "apple", "cherry", "banana"}
	expected := []string{"apple", "banana", "cherry"}
	flags := &FlagOpions{unique: true}
	sorts := NewSorts(flags, lines)
	sorts.Sort()
	for i, line := range sorts.lines {
		if line != expected[i] {
			t.Errorf("Unique() failed at index %d: got %s, want %s", i, line, expected[i])
		}
	}
}

func TestCheckSorted(t *testing.T) {
	sortedLines := []string{"apple", "banana", "cherry"}
	unsortedLines := []string{"banana", "apple", "cherry"}

	flags := &FlagOpions{}
	sorts := NewSorts(flags, sortedLines)
	if !sorts.sortCheck() {
		t.Errorf("checkSorted() failed: expected sortedLines to be sorted")
	}

	sorts = NewSorts(flags, unsortedLines)
	if sorts.sortCheck() {
		t.Errorf("checkSorted() failed: expected unsortedLines to be unsorted")
	}
}

func TestMonthMixed(t *testing.T) {
    lines := []string{"apple", "Jan", "Banana"}
    flags := &FlagOpions{month: true}
    sorts := NewSorts(flags, lines)
    sorts.Sort()
    want := []string{"Jan", "Banana", "apple"}
    if len(sorts.lines) != len(want) {
        t.Fatalf("unexpected length: got %d want %d", len(sorts.lines), len(want))
    }
    for i := range want {
        if sorts.lines[i] != want[i] {
            t.Errorf("TestMonthMixed: index %d got %q want %q", i, sorts.lines[i], want[i])
        }
    }
}

func TestNumericMixed(t *testing.T) {
    lines := []string{"abc", "2", "10"}
    flags := &FlagOpions{numb: true}
    sorts := NewSorts(flags, lines)
    sorts.Sort()
    want := []string{"2", "10", "abc"} 
    if len(sorts.lines) != len(want) {
        t.Fatalf("unexpected length: got %d want %d", len(sorts.lines), len(want))
    }
    for i := range want {
        if sorts.lines[i] != want[i] {
            t.Errorf("TestNumericMixed: index %d got %q want %q", i, sorts.lines[i], want[i])
        }
    }
}

func TestManyFlags(t *testing.T) {
	lines := []string{"2M\t", "500", "1K\t\t", "1G", "750K"}
	expected := []string{"1G", "2M", "750K", "1K", "500"}
	flags := &FlagOpions{human: true, blanks: true, unique: true, reverse: true}
	sorts := NewSorts(flags, lines)
	sorts.Sort()
	for i, line := range sorts.lines {
		if line != expected[i] {
			t.Errorf("ManyFlags() failed at index %d: got %s, want %s", i, line, expected[i])
		}
	}
}

func TestManyFlags2(t *testing.T) {
	lines := []string{"b\t2", "a\t3", "c\t1", "a\t3"}
	expected := []string{"c\t1", "b\t2", "a\t3"}
	flags := &FlagOpions{column: 1, unique: true, reverse: true}
	sorts := NewSorts(flags, lines)
	sorts.Sort()
	for i, line := range sorts.lines {
		if line != expected[i] {
			t.Errorf("ManyFlags2() failed at index %d: got %s, want %s", i, line, expected[i])
		}
	}
}


