package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"sync/atomic"
	"time"
)

func main() {
	timeout := flag.Int("timeout", 10, "timeout for connection to server")
	flag.Parse()
	args := flag.Args()
	if len(args) < 2 {
		fmt.Fprint(os.Stderr, "Usage: <host> <port> [--timeout=10s]\n")
		os.Exit(1)
	}
	address := args[0] + ":" + args[1]
	conn, err := net.DialTimeout("tcp", address, time.Second*time.Duration(*timeout)) // подклчюение с установленным таймаутом
	if err != nil {
		fmt.Fprintf(os.Stderr, "connection error: %v\n", err)
		os.Exit(2)
	}

	done := make(chan struct{}) // канал для ожидания горутин
	var connClosed atomic.Bool  // флаг закрытия соединения (atomic для безопасности в горутинах)

	defer func() {
		if !connClosed.Load() { // если соединение не закрыто, закрываем его
			err := conn.Close()
			if err != nil {
				fmt.Fprintf(os.Stderr, "error to close connection: %v", err)
				os.Exit(3)
			}
		}
	}()

	go func() {
		_, err := io.Copy(os.Stdout, conn)
		if err != nil && !connClosed.Load() {
			fmt.Fprintf(os.Stderr, "error to read from connection: %v", err)
		}
		if !connClosed.Load() {
			fmt.Println("Connection closed by foreign host.")
		}
		close(done) // сигнализируем о завершении
	}()

	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			text := scanner.Text()
			_, err := io.WriteString(conn, text+"\n")
			if err != nil && err != io.EOF {
				fmt.Fprintf(os.Stderr, "error to write from connection: %v", err)
				break
			}
			if err == io.EOF {
				break
			}
		}
		fmt.Println("Connection closed.")
		connClosed.Store(true) // устанавливаем флаг закрытия соединения
		err := conn.Close()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error to close connection: %v", err)
			os.Exit(3)
		} // закрываем соединение
	}()

	<-done // ждем завершения горутины чтения
}
