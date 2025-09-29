package app

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"
)

type FlagOpions struct {
	after  int
	before int
	circle int
	count  bool
	ignore bool
	verse  bool
	fix    bool
	number bool
}

type Grep struct {
	flags   *FlagOpions
	pattern string
	lines   []string
}

func Input(args []string) ([]string, error) {
	var scanner *bufio.Scanner
	if len(args) > 0 {
		file, err := os.Open(args[1])
		if err != nil {
			return nil, err
		}
		defer func() {
			err := file.Close()
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error closing file:", err)
			}
		}()
		scanner = bufio.NewScanner(file)
	} else {
		scanner = bufio.NewScanner(os.Stdin)
	}
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "Error reading input:", err)
		os.Exit(1)
	}
	return lines, nil
}

func ParseFlags() *FlagOpions {
	var flags FlagOpions
	flag.IntVar(&flags.after, "A", 0, "N lines after")
	flag.IntVar(&flags.before, "B", 0, "N lines before")
	flag.IntVar(&flags.circle, "C", 0, "N lines after & before (like -A N -B N)")
	flag.BoolVar(&flags.count, "c", false, "only number of lines which satisfy the pattern")
	flag.BoolVar(&flags.ignore, "i", false, "ignore case")
	flag.BoolVar(&flags.verse, "v", false, "reverse pattern (lines which doesnt satisfy)")
	flag.BoolVar(&flags.fix, "F", false, "exact substring match")
	flag.BoolVar(&flags.number, "n", false, "number of line before each found line")
	return &flags
}

func ExpandArgs(args []string) []string {
	var out []string
	needsValue := map[rune]bool{
		'A': true, // -A требует аргумент
		'B': true, // -B требует аргумент
		'C': true, // -C требует аргумент
	}

	for i := 0; i < len(args); i++ {
		a := args[i]
		// пропуск --флагов и одиночных флагов вида "-x"
		if !strings.HasPrefix(a, "-") || strings.HasPrefix(a, "--") || len(a) == 2 {
			out = append(out, a)
			continue
		}

		// разбор составных флагов вида "-abc" или "-A3"
		runes := []rune(a[1:])
		j := 0
		for j < len(runes) {
			r := runes[j]
			// если символ не буква оставляем как есть

			if needsValue[r] {
				// флаг требует значения
				// если в том же есть остаток это значение
				if j+1 < len(runes) {
					val := string(runes[j+1:])
					out = append(out, "-"+string(r))
					out = append(out, val)
				}
				break
			}

			// обычный булевый короткий флаг
			out = append(out, "-"+string(r))
			j++
		}
	}

	return out
}

func NewGrep(flags *FlagOpions, pattern string, lines []string) *Grep {

	return &Grep{
		flags:   flags,
		pattern: pattern,
		lines:   lines,
	}
}

// Filter выполняет основную логику фильтрации и вывода.
func (g *Grep) Filter() {
	// Применяем флаг -C
	if g.flags.circle > 0 {
		g.flags.after = g.flags.circle
		g.flags.before = g.flags.circle
	}

	// Находим индексы всех строк, соответствующих шаблону
	matchedIdxs := g.findMatches()

	// Если флаг -c, выводим количество и завершаем работу
	if g.flags.count {
		fmt.Println(len(matchedIdxs))
		return
	}

	// Формируем и выводим результат с учетом контекста
	g.printResult(matchedIdxs)
}

// findMatches ищет строки, соответствующие шаблону и возвращает их индексы
func (g *Grep) findMatches() []int {
	var matchedIdxs []int
	var matcher func(string) bool // функция для проверки соответствия строки шаблону (регулярка или фиксированная строка)

	pattern := g.pattern
	if g.flags.ignore {
		pattern = strings.ToLower(pattern)
	}

	if g.flags.fix {
		// Фиксированная строка
		matcher = func(line string) bool {
			if g.flags.ignore {
				line = strings.ToLower(line)
			}
			return strings.Contains(line, pattern)
		}
	} else {
		// Регулярное выражение
		if g.flags.ignore {
			pattern = "(?i)" + pattern // добавляем флаг игнорирования регистра
		}
		re, err := regexp.Compile(pattern)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error compiling regex: %v\n", err)
			os.Exit(1)
		}
		matcher = func(line string) bool {
			return re.MatchString(line)
		}
	}

	for i, line := range g.lines {
		match := matcher(line)
		// Инвертируем результат, если указан флаг -v
		if g.flags.verse {
			match = !match
		}
		if match {
			matchedIdxs = append(matchedIdxs, i)
		}
	}
	return matchedIdxs
}

// printResult выводит строки по их индексам с учетом контекста и других флагов
func (g *Grep) printResult(idxs []int) {
	if len(idxs) == 0 {
		return
	}

	linesToPrint := make(map[int]bool)

	for _, idx := range idxs {
		// Добавляем саму найденную строку
		linesToPrint[idx] = true

		// Контекст "до" (-B)
		start := idx - g.flags.before
		if start < 0 {
			start = 0
		}
		for i := start; i < idx; i++ {
			linesToPrint[i] = true
		}

		// Контекст "после" (-A)
		end := idx + g.flags.after
		if end >= len(g.lines) {
			end = len(g.lines) - 1
		}
		for i := idx + 1; i <= end; i++ {
			linesToPrint[i] = true
		}
	}

	for i := 0; i < len(g.lines); i++ {
		if linesToPrint[i] {
			if g.flags.number {
				fmt.Printf("%d:%s\n", i+1, g.lines[i])
			} else {
				fmt.Println(g.lines[i])
			}
		}
	}
}
