package main

import (
	"Cut/app"
	"flag"
	"fmt"
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

	grep := app.NewCut(flags, lines)
	grep.Cutter()
}
