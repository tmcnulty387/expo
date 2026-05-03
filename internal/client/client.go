// Package client models the internal state of an Expo client.
// Applications seeking to provide an Expo interface should create a [Client].
//
// Messages are communicated by:
// 1. Sending a valid message.Header.
// 2. Sending a stream of bytes, message.Header.Size bytes long.
//
// This stream of bytes can be deserialized into the type specified by
// MessageHeader.Kind.
package client

// TODO: Command struct that can be used for CLI/GUI to communicate with its
// Client?

type Client struct {
	// TODO: fields?
}
