package socketeer

import (
	"bytes"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)


var samplebuffer = new(bytes.Buffer)

// Id generation
type sample struct {}

func (s *sample) GetUniqueId() string{
	return "user" + strconv.Itoa(int(time.Now().UnixNano()))
}

func (s *sample) Run(socketerr *SocketeerManager){
	tick := time.NewTicker(5 * time.Second)
	for{
		<-tick.C
		socketerr.Broadcast([]byte("hello"))
	}
}

func (s *sample) OnMessage(message []byte , sendChannels map[string]chan []byte){

}



func BootstrapRouter() *http.ServeMux{


	manager := SocketeerManager{
		IdGen: &sample{},
	}

	manager.AddDispatcher(&sample{})
	manager.AddMessageHandler(&sample{})
	manager.AddOnConnectHook(func(manager *SocketeerManager , connectionId string) {
		fmt.Fprint(samplebuffer , "connected")
	})
	manager.Init()

	mux:= http.NewServeMux()
	mux.HandleFunc("/" , func(writer http.ResponseWriter, request *http.Request) {
		id , _ := manager.Manage(writer, request)
		manager.SendToId(id , []byte(id))
	})

	return mux
}









func TestSocketeerManager_main(t *testing.T){

	t.Run("Recieve connectionId " , func(t *testing.T) {
		server := httptest.NewServer(BootstrapRouter())

		wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/"
		conn ,_ , err := websocket.DefaultDialer.Dial(wsURL , nil)

		defer conn.Close()

		if err != nil{
			t.Fatalf("Couldnt connect to websocket : %s" , err.Error())
		}

		_ , message , err := conn.ReadMessage()

		if err != nil{
			t.Fatal("Couldnt read message from the websocket")
		}

		assert.Greater(t , len(message) , 0)

		time.Sleep(11 * time.Second)
		_ , helloMessage , err := conn.ReadMessage()

		assert.Equal(t , []byte("hello") , helloMessage)

		value , _ := ioutil.ReadAll(samplebuffer)

		assert.Equal(t ,[]byte("connected") , value)

	})
}