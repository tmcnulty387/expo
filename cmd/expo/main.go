package main

import (
	"context"
	"crypto/tls"
	"flag"
	"log"
	"os"
	"sync"
	"time"

	"gioui.org/app"

	"github.com/Go-20255/team-project-malloc4.git/internal/ui"
)

// TODO: Separate out into internal/networking? Replace with CLI function?
func cli(ctx context.Context) error {
	// TODO: remove placeholder example connection attempt
	conn, err := tls.Dial("tcp", "mail.google.com:443", nil)
	if err != nil {
		return err
	}
	log.Printf("connection: %s\n", conn.RemoteAddr())
	defer conn.Close()
	select {
	case <-ctx.Done():
		return ctx.Err()
	// TODO: Remove placeholder network wait
	case <-time.After(5 * time.Second):
	}
	return nil
}

// High-level client logic.
// Should be ran in a goroutine to accomodate gioui requirements.
// This function does not return, but will call os.Exit directly to terminate
// the program.
func run(headless bool) {
	exitCode := 0
	ctx, cancel := context.WithCancel(context.Background())
	// Tasks should write to this channel to signal that the client should quit.
	quit := make(chan error, 1)
	var tasks sync.WaitGroup

	//
	// TODO: Integrate properly into a CLI for networking? Provide channels
	// so that the GUI can signal networking requests etc.
	//
	tasks.Go(func() { quit <- cli(ctx) })

	//
	// Launch GUI loop.
	//
	if !headless {
		tasks.Go(func() { quit <- ui.Loop(ctx, new(app.Window)) })
	}

	//
	// Block until one of the tasks exits.
	//
	err := <-quit
	if err != nil {
		log.Println(err)
		exitCode = 1
	}
	log.Println("Shutting down")
	cancel()
	tasks.Wait()
	os.Exit(exitCode)
}

func main() {
	var headless bool
	flag.BoolVar(&headless, "headless", false, "Launch without GUI")
	flag.Parse()

	go run(headless)

	app.Main()
}
