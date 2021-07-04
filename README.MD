## Socketeer ![](https://github.com/kiishi/socketeer/workflows/CI/badge.svg)

> ( pronounced "socket-eer" )

An little abstraction on top of gorilla's *[Websocket Library](https://github.com/gorilla/websocket)* \

### Why i build this?

I found myself doing the same thing over and over again whenever I work with websockets... So I messed around and built
this✌🏽

### Install

To add this to your project, run

```shell
go get github.com/kiishi/socketeer
```

### Usage

Using this means you avoid manually handling `*websocket.Conn`

sample usage

```go
package main

import (
	"github.com/kiishi/socketeer"
	"net/http"
	"strconv"
	"time"
	"log"
)

var sManager = socketeer.Manager{
	/*
		This factory function should return a unique string
		which would be used to identify every websocket connection
	*/
	IdGen: func() string {
		return strconv.Itoa(int(time.Now().UnixNano()))
	},
}

type WsHandler struct {
	allConnections map[string]interface{}
}

func (w *WsHandler) OnMessage(manager *socketeer.Manager, ctx *socketeer.MessageContext) {
//    This gets run on every WS message sent
}

func (w *WsHandler) OnConnect(manager *socketeer.Manager, request *http.Request, connectionId string) {
	//	This gets run on every websocket connection
}

func (w *WsHandler) OnDisconnect(manager *socketeer.Manager, connectionId string) {
	//    this gets run on every websocket disconnection.
}

func main() {
	sManager.AddMessageHandler(&WsHandler{})
	sManager.Init()

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", func(writer http.ResponseWriter, request *http.Request) {
		_, err := sManager.Manage(writer, request)
		if err != nil {
			writer.Write([]byte("close"))
		}
	})

	log.Println("Starting server...")
	err := http.ListenAndServe(":8080", mux)

	if err != nil {
		log.Println("server closed")
	}
}
```

### Examples

See *[Example](https://github.com/kiishi/socketeer/tree/master/example)*
