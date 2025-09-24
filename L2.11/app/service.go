package app

import (
	"sort"
	"strings"
)

func AnSearcher(s []string) map[string][]string {
	groups := make(map[string][]string) // ключ - отсортированные буквы, значение - слайс слов, которые являются анаграммами

	// группируем слова по их отсортированным буквам

	for _, elem := range s {
		elem = strings.ToLower(elem)
		key := sortRunes(elem)
		groups[key] = append(groups[key], elem)
	}

	result := make(map[string][]string)

	// отбираем только те группы, которые содержат более одного слова
	for _, elem := range groups {
		if len(elem) > 1 {
			result[elem[0]] = elem
			sort.Strings(result[elem[0]])
		}
	}

	return result
}

func sortRunes(s string) string {
	runes := []rune(s)
	sort.Slice(runes, func(i, j int) bool {
		return runes[i] < runes[j]
	})
	return string(runes)
}
