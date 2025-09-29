package main

import (
	"flag"
	"fmt"
	"grepGo/app"
	"os"
)

func main() {
	flags := app.ParseFlags()               // определить флаги командной строки
	expanded := app.ExpandArgs(os.Args[1:]) // развернуть объединённые короткие флаги и распарсить их
	if err := flag.CommandLine.Parse(expanded); err != nil {
		fmt.Fprintln(os.Stderr, "flag parse error:", err)
		os.Exit(2)
	}

	lines, err := app.Input(flag.Args()) // считать входные данные из файла или stdin

	if err != nil {
		fmt.Fprintln(os.Stderr, "Error reading input:", err)
		os.Exit(1)
	}

	if len(flag.Args()) == 0 {
		fmt.Fprintln(os.Stderr, "Error: pattern not provided")
		os.Exit(1)
	}
	pattern := flag.Args()[0] // получить шаблон поиска из аргументов командной строки

	grep := app.NewGrep(flags, pattern, lines) // создать структуру grep с флагами и входными данными
	grep.Filter()                              // выполнить фильтрацию
}
