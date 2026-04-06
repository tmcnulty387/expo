// Common networking code between client and server.

package expo

import "encoding/json"

const ServerPort int = 56469

//
// Messages between Expo clients and servers are encoding in JSON and
// structured like so:
//
//    {
//    	"type": string,
//    	"body": {
//    	  ...
//    	},
//    }
//
// Clients should first unmarshal the "type", and then unmarshal the
// "body" into an appropriately typed struct.
//
// Note that Messages are only intended to be communicated once a
// WebSocket connection has been established, not as the body to any
// other HTTP request.

type Message struct {
	Type string          `json:"type"`
	Body json.RawMessage `json:"body"`
}

const (
	MessageTypeJoinRoom string = "JoinRoom"
)

type JoinRoom struct {
	RoomCode string `json:"room_code"`
}
