package main

import (
	"NtpWb/app"
	"log"
	"fmt"
	"os"
)

func main() {
	// Получаем время через NTP
	time, err := app.GetTime()


	// Если произошла ошибка, выводим её в stderr и завершаем программу с кодом 1
	if err != nil {
		// Логируем ошибку в stderr
		log.Fatalf("Error: %v", err)
		// Завершаем с ненулевым кодом (ошибка)
		os.Exit(1)
	}

	// Если ошибок нет, выводим время в STDOUT
	fmt.Println(time)
}