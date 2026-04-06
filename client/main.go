package main

import (
	"bufio"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
)

var (
	TIMESWAIT    = 0
	TIMESWAITMAX = 5
	in           = bufio.NewReader(os.Stdin)
)

func getInput(input chan string) {
	result, err := in.ReadString('\n')
	if err != nil {
		log.Println(err)
		return
	}
	input <- result
}

func main() {
	if len(os.Args) != 2 {
		fmt.Printf("Usage: %s server:port\n", os.Args[0])
		return
	}

	host := os.Args[1]

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	input := make(chan string, 1)
	go getInput(input)

	url := url.URL{Scheme: "ws", Host: host, Path: "/connect"}
	c, _, err := websocket.DefaultDialer.Dial(url.String(), nil)
	if err != nil {
		log.Fatalln(err)
	}
	defer c.Close()

	log.Println("Launching Expo client app")
	go startApp()

	// TODO: Remove/integrate into messages to be passed between main expo app?
	log.Println("Launching example websocket client echo code")

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("ReadMessage() error:", err)
				return
			}
			log.Printf("Received: %s", message)
		}
	}()

	for {
		select {
		case <-time.After(4 * time.Second):
			log.Println("Please give me input!", TIMESWAIT)
			TIMESWAIT++
			if TIMESWAIT > TIMESWAITMAX {
				syscall.Kill(syscall.Getpid(), syscall.SIGINT)
			}
		case <-done:
			return
		case t := <-input:
			err := c.WriteMessage(websocket.TextMessage, []byte(t))
			if err != nil {
				log.Println("Write error:", err)
				return
			}
			TIMESWAIT = 0
			go getInput(input)
		case <-interrupt:
			log.Println("Caught interrupt signal - quitting!")
			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("Write close error:", err)
				return
			}
			select {
			case <-done:
			case <-time.After(2 * time.Second):
			}
			return
		}
	}
}
