package main

import (
	"github.com/kiishi/socketeer"
	"log"
	"net/http"
	"strconv"
	"time"
)

var store = ChatStoreObject{
	AllUsers: map[string]interface{}{},
	Rooms:    map[string][]string{},
}

var managers = socketeer.Manager{
	IdGen: func() string {
		return strconv.Itoa(int(time.Now().UnixNano()))
	},
}


func WsHandler(w http.ResponseWriter, r *http.Request) {
	_, err := managers.Manage(w, r)
	if err != nil {
		w.Write([]byte("close"))
	}
}

func main() {
	managers.AddMessageHandler(&store)
	managers.Init()

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", WsHandler)

	log.Println("Starting server...")
	err := http.ListenAndServe(":8080", mux)

	if err != nil {
		log.Println("server closed")
	}
}
