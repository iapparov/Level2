package app

import (
	"testing"
)

func TestAnSearcher(t *testing.T) {
	tests := []struct {
		input    []string
		expected map[string][]string
	}{
		{
			input: []string{"пятак", "пятка", "тяпка", "листок", "слиток", "столик", "стол"},
			expected: map[string][]string{
				"пятак": {"пятак", "пятка", "тяпка"},
				"листок": {"листок", "слиток", "столик"},
			},
		},
		{
			input: []string{"кот", "ток", "окт", "дом", "мод"},
			expected: map[string][]string{
				"кот": {"кот", "окт", "ток"},
				"дом": {"дом", "мод"},
			},
		},
		{
			input: []string{"hello", "world"},
			expected: map[string][]string{},
		},
		{
			input: []string{},
			expected: map[string][]string{},
		},
		}

	for _, test := range tests {
		result := AnSearcher(test.input)
		if len(result) != len(test.expected) {
			t.Errorf("For input %v, expected %v, but got %v", test.input, test.expected, result)
			continue
		}
		for k, v := range test.expected {
			resSlice, ok := result[k]
			if !ok {
				t.Errorf("For input %v, expected key %v not found in result %v", test.input, k, result)
				continue
			}
			if len(resSlice) != len(v) {
				t.Errorf("For input %v, expected value length %d for key %v, but got %d", test.input, len(v), k, len(resSlice))
				continue
			}
			for i := range v {
				if resSlice[i] != v[i] {
					t.Errorf("For input %v, expected value %v for key %v, but got %v", test.input, v, k, resSlice)
					break
				}
			}
		}		
	}
}

func TestSortRunes(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"пятак", "акптя"},
		{"листок", "иклост"},
		{"hello", "ehllo"},
		{"world", "dlorw"},
		{"", ""},
	}

	for _, test := range tests {
		result := sortRunes(test.input)
		if result != test.expected {
			t.Errorf("For input %v, expected %v, but got %v", test.input, test.expected, result)
		}
	}
}