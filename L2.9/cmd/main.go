package main

import (
	"StrUnpacker/app"
	"bufio"
	"fmt"
	"log"
	"os"
)

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	var s string
	if scanner.Scan() {
		s = scanner.Text()
	}

	// Вызываем функцию распаковки строки
	ans, err := app.StrUnpack(s)


	// Если произошла ошибка, выводим её в stderr и завершаем программу с кодом 1
	if err != nil {
		// Логируем ошибку в stderr
		log.Fatalf("Error: %v", err)
		// Завершаем с ненулевым кодом (ошибка)
		os.Exit(1)
	}

	// Если ошибок нет, выводим в STDOUT
	fmt.Println(ans)
}