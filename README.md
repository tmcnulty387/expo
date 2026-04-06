# EXPO

Made as part of CSCI 541 - Concurrent and Systems Programming in Go at RIT, Spring 2026.

A native, distributed digital whiteboard application.

`Please see docs/Proposal.md for the detailed Project Proposal.`

*__Team Members:__ Sebastian LaVine, Jane Majewski, Tim McNulty, Rina Peshori*
=======
## Intended Components:
- Gio UI
- Standard Go TCP/IP stack

## MVP (checkpoint 1):
- Bitmap image data communicated in real time between 2 clients.
- Pixel-based drawing tools only, implemented using bitmap


## Build Instructions

This repository contains two separate programs: the Expo client, and the
Expo server.

To build and run the client:

```shell-session
$ go run ./client 
```

To build and run the server:

```shell-session
$ go run ./server 
```


Live reload with Air:
```shell-session
$ air --build.cmd "go build -o ./tmp/expo-server ./server" \
    --build.entrypoint "./tmp/expo-server"
```
=======
## Testing Goals:
It works :) Tested using automated unit tests.
- Network failures handled (resiliency)
- Security, authentication
- Latency
