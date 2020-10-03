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

type sampleDispatcher struct{}

func (s *sampleDispatcher) Run(socketerr *Manager) {
	tick := time.NewTicker(5 * time.Second)
	for {
		<-tick.C
		socketerr.Broadcast([]byte("hello"))
	}
}

func BootstrapRouter() *http.ServeMux {
	manager := Manager{
		IdGen: func() string {
			return "user" + strconv.Itoa(int(time.Now().UnixNano()))
		},
		OnConnect: func(manager *Manager, request *http.Request, connectionId string) {
			fmt.Fprint(samplebuffer, "connected")
		},
	}

	manager.AddDispatcher(&sampleDispatcher{})

	manager.Init()

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		id, _ := manager.Manage(writer, request)
		manager.SendToId(id, []byte(id))
	})

	return mux
}

func ConnectToTestServer(t *testing.T , server *httptest.Server) *websocket.Conn {

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Couldnt connect to websocket : %s", err.Error())
	}
	return conn
}

func TestSocketeerManager_main(t *testing.T) {
	server := httptest.NewServer(BootstrapRouter())

	t.Run("Recieves connectionId ", func(t *testing.T) {
		conn := ConnectToTestServer(t, server)

		_, message, err := conn.ReadMessage()
		if err != nil {
			t.Fatal("Couldnt read message from the websocket")
		}
		assert.Greater(t, len(message), 0)
		time.Sleep(11 * time.Second)
		_, helloMessage, err := conn.ReadMessage()

		assert.Equal(t, []byte("hello"), helloMessage)

		// OnConnectMessage Ran
		value, _ := ioutil.ReadAll(samplebuffer)
		assert.Equal(t, []byte("connected"), value)
	})

}
