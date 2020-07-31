package main

import (
	"fmt"
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
	OnConnect: func(m *socketeer.Manager , request *http.Request , connectionId string){
		m.SendToId(connectionId, []byte(connectionId))
		store.AllUsers[connectionId] = nil
		log.Println(request.URL.Path)
	},
	OnDisconnect: func(m* socketeer.Manager , connectionId string ){
		fmt.Printf("%s disconnected" , connectionId)
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
