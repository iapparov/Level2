package app

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sort"
	"flag"
	"strconv"
)

type FlagOpions struct {
	column int
	numb bool
	reverse bool
	unique bool
	month bool
	blanks bool
	checkSort bool
	human bool
}

type Sorts struct {
	flags *FlagOpions
	lines []string
}

func Input(args []string) ([]string, error){
	var scanner *bufio.Scanner
	if len(args)>0{
		file, err := os.Open(args[0])
		if err != nil{
			return nil, err
		}
		defer func () {
			err := file.Close()
			if err != nil{
				fmt.Fprintln(os.Stderr, "Error closing file:", err)
			}
		}()
		scanner = bufio.NewScanner(file)
	}else{
		scanner = bufio.NewScanner(os.Stdin)
	}
	var lines []string
	for scanner.Scan(){
		lines = append(lines, scanner.Text())
	}

		
	if err := scanner.Err(); err != nil{
		fmt.Fprintln(os.Stderr, "Error reading input:", err)
		os.Exit(1)
	}
	return lines, nil
}

func ParseFlags() *FlagOpions{
	var flags FlagOpions
	flag.IntVar(&flags.column, "k", 0, "sort by column N")
	flag.BoolVar(&flags.numb, "n", false, "sort by numbs")
	flag.BoolVar(&flags.reverse, "r", false, "sort in reverse")
	flag.BoolVar(&flags.unique, "u", false, "only unique str")
	flag.BoolVar(&flags.month, "M", false, "sort by month")
	flag.BoolVar(&flags.blanks, "b", false, "ignore trailing blanks")
	flag.BoolVar(&flags.checkSort, "c", false, "check if sort")
	flag.BoolVar(&flags.human, "h", false, "sort by human readable numbers")
	return &flags
}

// ExpandArgs разворачивает объединённые короткие флаги типа "-uk2" или "-uk 2" в отдельные аргументы:
// "-uk 2" -> ["-u", "-k", "2"]
// "-uk2"  -> ["-u", "-k", "2"]
// Поддерживает указание, какие короткие флаги требуют значения (здесь -k)
func ExpandArgs(args []string) []string {
    var out []string
    needsValue := map[rune]bool{
        'k': true, // -k требует аргумент
    }

    for i := 0; i < len(args); i++ {
        a := args[i]
        // пропуск --флагов и одиночных флагов вида "-x"
        if !strings.HasPrefix(a, "-") || strings.HasPrefix(a, "--") || len(a) == 2 {
            out = append(out, a)
            continue
        }

        // короткий флаг с несколькими символами, например "-uk2" или "-abc"
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

func NewSorts (flags *FlagOpions, lines []string) *Sorts{
	return &Sorts{
		flags: flags,
		lines: lines,
	}
}

func (s *Sorts) Sort(){

	if s.flags.checkSort{ // проверка отсортированности
		if !s.sortCheck(){
			fmt.Println("Input is not sorted")
		}
		return
	}

	if s.flags.blanks { // игнорирование пробелов в конце строк
		s.sortBlanks()
	}

	sort.SliceStable(s.lines, func(i, j int) bool { // стабильная сортировка для сохранения порядка при равенстве
		less := s.compare(s.lines[i], s.lines[j])
		if s.flags.reverse {
			return !less
		}
		return less
	})

	if s.flags.unique { // удаление дубликатов после сортировки
		s.unique()
	}

	for _, line := range s.lines { // вывод результата
		fmt.Println(line)
	}
}

func (s *Sorts) unique(){
	if len(s.lines) == 0 {
		return
	}
	var result []string
	last := s.lines[0]
	result = append(result, last)
	for _, line := range s.lines[1:] {
		if line != last {
			result = append(result, line)
			last = line
		}
	}
	s.lines = result
}

func (s *Sorts) sortBlanks() {
	for i, line := range s.lines{
		s.lines[i] = strings.TrimRight(line, " \t")
	}
}

func (s* Sorts) sortCheck() bool{
	return sort.SliceIsSorted(s.lines, func(i, j int) bool {
		return s.compare(s.lines[i], s.lines[j])
	})
}

func (s *Sorts) compare(a, b string) bool {

    // извлечение ключа для сравнения
    keyA := s.extractKey(a)
    keyB := s.extractKey(b)

    // всегда убираем внешние пробелы перед сравнениями
    keyAT := strings.TrimSpace(keyA)
    keyBT := strings.TrimSpace(keyB)

    // сравнение в зависимости от флагов

    switch {
    case s.flags.month:
        // monthMap нечувствителен к регистру
        var monthMap = map[string]int{
            "jan": 1, "feb": 2, "mar": 3, "apr": 4,
            "may": 5, "jun": 6, "jul": 7, "aug": 8,
            "sep": 9, "oct": 10, "nov": 11, "dec": 12,
        }

		// если оба ключа месяцы, сравниваем по номеру месяца
        ma, oka := monthMap[strings.ToLower(keyAT)] 
        mb, okb := monthMap[strings.ToLower(keyBT)]  
        if oka && okb {
            return ma < mb
        } 
		
		// если только один из ключей месяц, он считается "меньше" не месяца
        if oka != okb {
            return oka
        }

		// оба не месяца строковое сравнение
        return keyAT < keyBT

    case s.flags.human:
        return parseHuman(keyAT) < parseHuman(keyBT)

    case s.flags.numb:
        numA, errA := strconv.ParseFloat(keyAT, 64)
        numB, errB := strconv.ParseFloat(keyBT, 64)
        // если оба парсятся обычное сравнение
        if errA == nil && errB == nil {
            return numA < numB
        }
        // если один парсится, он считается "меньше" нечислового (как в приведённых реализациях раньше)
        if errA == nil && errB != nil {
            return true
        }
        if errA != nil && errB == nil {
            return false
        }
        // оба не парсятся строковое сравнение
        return keyAT < keyBT

    default:
        return keyAT < keyBT // сравнение без крайних пробелов
    }
}

func (s *Sorts) extractKey(line string) string {
	if s.flags.column > 0 { // извлечение по колонке (1-индексация)
		cols := strings.Split(line, "\t")
		if s.flags.column-1 < len(cols) {
			return cols[s.flags.column-1]
		}
	}
	return line
}

func parseHuman(s string) float64 {
	
	if s == "" {
		return 0
	}
	var humanSuffix = map[string]float64{ // множители для суффиксов человекочитаемых размеров
		"K": 1024,
		"M": 1024 * 1024,
		"G": 1024 * 1024 * 1024,
		"T": 1024 * 1024 * 1024 * 1024,
	}
	last := s[len(s)-1:] // последний символ
	numStr := s // числовая часть
	multiplier := 1.0 // множитель по умолчанию
	if val, ok := humanSuffix[last]; ok { // если есть суффикс, отделяем его
		numStr = s[:len(s)-1]
		multiplier = val
	}
	num, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0
	}
	return num * multiplier
}