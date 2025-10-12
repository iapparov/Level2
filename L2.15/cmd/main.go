package main

import (
	"UnixShell/app"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGINT)
	go app.SigCancel(sigc)

	app.UnixShell()
}
