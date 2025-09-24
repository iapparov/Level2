package main

import (
	"Dictionary/app"
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

func main() {
	s := ImportStr()
	res := app.AnSearcher(s)
	for k, v := range res {
		fmt.Println(k, ":", v)
	}
}

func ImportStr() []string {
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n') // читаем до Enter
	if err != nil && err != io.EOF {
		fmt.Fprintln(os.Stderr, "error reading input:", err)
		os.Exit(1)
	}

	// разбиваем по пробелам
	words := strings.Fields(line)
	return words
}
