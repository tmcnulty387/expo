# EXPO

Made as part of CSCI 541 - Concurrent and Systems Programming in Go at RIT, Spring 2026.

A native, distributed digital whiteboard application.

`Please see docs/Proposal.md for the detailed Project Proposal.`

*__Team Members:__ Sebastian LaVine, Jane Majewski, Tim McNulty, Rina Peshori*
=======
## Intended Components:
- Gio UI
- Standard Go TCP/IP stack (TLS)

## MVP (checkpoint 1):
- Bitmap image data communicated in real time between 2 clients.
- Pixel-based drawing tools only, implemented using bitmap


## Build Instructions

This repository contains one program which implements the Expo GUI and
networking protocol.

To build and run:

```shell-session
$ go run ./cmd/expo
```


Live reload with Air:
```shell-session
$ air --build.cmd "go build -o ./tmp/expo ./cmd/expo" \
    --build.entrypoint "./tmp/expo"
```
=======
## Testing Goals:
It works :) Tested using automated unit tests.
- Network failures handled (resiliency)
- Security, authentication
- Latency
