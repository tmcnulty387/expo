// Expo server.
// Receives connections from Expo clients and coordinates drawing sessions.

package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/Go-20255/team-project-malloc4/expo"

	"github.com/gorilla/websocket"
)

var (
	clients   = make(map[*websocket.Conn]struct{})
	clientsMu sync.Mutex
	upgrader  = websocket.Upgrader{
		// TODO: Strout has this in the example code. Necessary?
		//CheckOrigin: func(r *http.Request) bool { return true },
	}
)

// Heartbeat sends a system message to all clients every X seconds
// func startHeartbeat(interval int) {
// 	ticker := time.NewTicker(time.Duration(interval) * time.Second)
// 	for range ticker.C {
// 		clientsMu.Lock()
// 		count := len(clients)
// 		clientsMu.Unlock()
//
// 		statusMsg := fmt.Sprintf("SYSTEM: Server is healthy. Active users: %d", count)
// 		broadcast([]byte(statusMsg))
// 	}
// }

func main() {
	// TODO: keep a heartbeat for keeping track of stray connections.
	// go startHeartbeat(cfg.Server.UpdateInterval)

	// TODO: provide a default / and /index.html landing page, and 404 page,
	// for wayward and confused clients
	// http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
	// 	http.ServeFile(w, r, "index.html")
	// 	http.ServeContent
	// })

	// Route for incoming client connections.
	http.HandleFunc("/connect", clientConnection)

	port := expo.ServerPort
	addr := fmt.Sprintf(":%d", port)

	log.Printf("Listening on port %d\n", port)
	err := http.ListenAndServe(addr, nil)
	if err != nil {
		log.Fatal(err)
	}
}

func clientConnection(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer ws.Close()

	// Add new client to clients set
	clientsMu.Lock()
	clients[ws] = struct{}{}
	clientsMu.Unlock()

	for {
		_, msg, err := ws.ReadMessage()
		if err != nil {
			log.Printf("User disconnected: %v", err)
			clientsMu.Lock()
			delete(clients, ws)
			clientsMu.Unlock()
			break
		}

		// get the IP:Port of the person who just typed
		clientIP := ws.RemoteAddr().String()

		// create a new message string: "127.0.0.1:54321 says: hello"
		formattedMsg := fmt.Sprintf("[%s]: %s", clientIP, string(msg))

		// broadcast to everyone
		broadcast([]byte(formattedMsg))
	}
}

func broadcast(msg []byte) {
	clientsMu.Lock()
	defer clientsMu.Unlock()

	now := time.Now().Format("15:04:05")
	timeStampedMsg := fmt.Sprintf("[%s] %s", now, string(msg))

	for client := range clients {
		err := client.WriteMessage(websocket.TextMessage, []byte(timeStampedMsg))
		if err != nil {
			log.Printf("Write error: %v", err)
			client.Close()
			delete(clients, client)
		}
	}
}
