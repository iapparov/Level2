package app

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type FlagOptions struct {
	fields    flagFields
	delimiter string
	separated bool
}

type flagField struct {
	start int
	end   int
}

type flagFields []flagField

type Cut struct {
	flag  *FlagOptions
	lines []string
}

func NewCut(flag *FlagOptions, lines []string) *Cut {
	return &Cut{
		flag:  flag,
		lines: lines,
	}
}

func (fr *flagFields) String() string {
	return fmt.Sprint(*fr)
}

func (fr *flagFields) Set(s string) error { // парсим флаг -f
	parts := strings.Split(s, ",") // делим на части по запятой
	for _, p := range parts {
		if strings.Contains(p, "-") {
			// диапазон
			bounds := strings.SplitN(p, "-", 2) //делим на две части по дефису
			startStr, endStr := bounds[0], bounds[1]

			var start, end int
			var err error

			if startStr == "" {
				start = 1 // cut считает от 1
			} else {
				start, err = strconv.Atoi(startStr)
				if err != nil {
					return fmt.Errorf("invalid range start: %s", startStr)
				}
			}

			if endStr == "" {
				end = -1 // "до конца"
			} else {
				end, err = strconv.Atoi(endStr)
				if err != nil {
					return fmt.Errorf("invalid range end: %s", endStr)
				}
			}

			*fr = append(*fr, flagField{start: start, end: end})
		} else {
			// одиночное поле
			num, err := strconv.Atoi(p)
			if err != nil {
				return fmt.Errorf("invalid field: %s", p)
			}
			*fr = append(*fr, flagField{start: num, end: num})
		}
	}
	return nil
}

func (fr flagFields) ShouldKeep(n int) bool {
	for _, f := range fr {
		if f.end == -1 { // до конца
			if n >= f.start {
				return true
			}
		} else {
			if n >= f.start && n <= f.end {
				return true
			}
		}
	}
	return false
}

func Input(args []string) ([]string, error) {
	var scanner *bufio.Scanner
	if len(args) > 0 {
		file, err := os.Open(args[0])
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

func ParseFlags() *FlagOptions {
	var flags FlagOptions
	flag.Var(&flags.fields, "f", "Number of fields")
	flag.StringVar(&flags.delimiter, "d", "\t", "Use another char. Default Tab")
	flag.BoolVar(&flags.separated, "s", false, "Only lines which contains separator")
	return &flags
}

func ExpandArgs(args []string) []string {
	var out []string
	needsValue := map[rune]bool{
		'f': true, // -f требует аргумент
		'd': true, // -d требует аргумент
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

func (c *Cut) Cutter() {
	for _, line := range c.lines {
		if c.flag.separated && !strings.Contains(line, c.flag.delimiter) {
			continue
		}

		fields := strings.Split(line, c.flag.delimiter)
		var selected []string

		for i, f := range fields {
			fieldNum := i + 1
			if c.flag.fields.ShouldKeep(fieldNum) {
				selected = append(selected, f)
			}
		}

		// всегда печатаем (если -s отфильтрованы, мы сюда уже не попадаем)
		fmt.Println(strings.Join(selected, c.flag.delimiter))
	}
}
