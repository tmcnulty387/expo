package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"iter"
	"log"
	"os"
	"strings"
	"sync"

	"gioui.org/app"

	"github.com/Go-20255/team-project-malloc4/internal/client"
	"github.com/Go-20255/team-project-malloc4/internal/ui"
)

type Options struct {
	Headless       bool
	LaunchCommands []string
}

type Transition int

const (
	Continue Transition = iota
	Quit
)

// Processes a command and interacts with the client.
// Returns Transition to signal what the caller should do after processing.
func processCommand(line string, client *client.Client) Transition {
	pullWord, stop := iter.Pull(strings.SplitSeq(line, " "))
	defer stop()
	command, ok := pullWord()
	if !ok {
		return Continue
	}
	switch command {
	case "quit":
		return Quit
	case "listen":
		address, ok := pullWord()
		if !ok {
			fmt.Println("Usage: listen address")
		} else {
			// TODO: Implement
			fmt.Println("TODO: Implement listen call")
			_ = address
		}
	}

	return Continue
}

// TODO: Separate out into internal/networking? Replace with CLI function?
func cli(ctx context.Context, launchCommands []string) error {
	// TODO: remove placeholder example connection attempt
	conn, err := tls.Dial("tcp", "mail.google.com:443", nil)
	if err != nil {
		return err
	}
	log.Printf("connection: %s\n", conn.RemoteAddr())
	defer conn.Close()

	lines := make(chan string, len(launchCommands)+10)
	go func() {
		for _, command := range launchCommands {
			lines <- strings.TrimSpace(command)
		}
		scanner := bufio.NewScanner(os.Stdin)
		for {
			fmt.Print("> ")
			if !scanner.Scan() {
				break
			}
			lines <- strings.TrimSpace(scanner.Text())
		}
	}()

	client := client.Client{}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case line := <-lines:
			switch processCommand(line, &client) {
			case Continue:
				continue
			case Quit:
				return nil
			}
		}
	}
}

// High-level client logic.
// Should be ran in a goroutine to accomodate gioui requirements.
// This function does not return, but will call os.Exit directly to terminate
// the program.
func run(options Options) {
	exitCode := 0
	ctx, cancel := context.WithCancel(context.Background())
	// Tasks should write to this channel to signal that the client should quit.
	quit := make(chan error, 1)
	var tasks sync.WaitGroup

	//
	// TODO: Integrate properly into a CLI for networking? Provide channels
	// so that the GUI can signal networking requests etc.
	//
	tasks.Go(func() { quit <- cli(ctx, options.LaunchCommands) })

	//
	// Launch GUI loop.
	//
	if !options.Headless {
		tasks.Go(func() { quit <- ui.Loop(ctx) })
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
	var options Options
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "expo - A decentralized digital whiteboard application.")

		fmt.Fprintln(os.Stderr, "\nUsage:")
		flag.PrintDefaults()
		fmt.Fprintln(os.Stderr, "\nArguments will be provided as launch commands to the command-line interface.")
	}
	flag.BoolVar(&options.Headless, "headless", false, "Launch without GUI")
	flag.Parse()

	options.LaunchCommands = flag.Args()

	go run(options)

	app.Main()
}
