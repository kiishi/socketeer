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

type MockObject struct {
	DisconnectBuffer *bytes.Buffer
	ConnectBuffer    *bytes.Buffer
	OnMessageBuffer  *bytes.Buffer
}

func (m *MockObject) OnConnect(manager *Manager, req *http.Request, connectionId string) {
	fmt.Fprint(m.ConnectBuffer, "[Connected]")
}

func (m *MockObject) OnDisconnect(manager *Manager, connectionId string) {
	fmt.Fprint(m.DisconnectBuffer, "[Disconnected]")
}

func (m *MockObject) OnMessage(manager *Manager, msg *MessageContext) {
	fmt.Fprint(m.OnMessageBuffer, "[Message]")
}



func BootStrapRouterWithCustomHandler(handlerObject *MockObject) *http.ServeMux {
	manager := Manager{
		IdGen: func() string {
			return "user" + strconv.Itoa(int(time.Now().UnixNano()))
		},
	}

	manager.AddMessageHandler(handlerObject)

	manager.AddDispatcher(&sampleDispatcher{})

	manager.Init()

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		id, _ := manager.Manage(writer, request)
		manager.SendToId(id, []byte(id))
	})

	return mux
}

func BootstrapRouterWithoutCustomHandler(mockObject *MockObject) *http.ServeMux {

	manager := Manager{
		IdGen: func() string {
			return "user" + strconv.Itoa(int(time.Now().UnixNano()))
		},
	}

	manager.onConnect = func(manager *Manager, request *http.Request, s string) {
		fmt.Fprint(mockObject.ConnectBuffer, "connected")

	}

	manager.onDisconnect = func(manager *Manager, s string) {
		fmt.Fprint(mockObject.DisconnectBuffer, "connected")

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

func ConnectToTestServer(t *testing.T, server *httptest.Server) (*websocket.Conn, *http.Response) {

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/"
	conn, response, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Couldnt connect to websocket : %s", err.Error())
	}
	return conn, response
}

func TestSocketeerManager_main(t *testing.T) {
	var MockHandlerObject = MockObject{
		new(bytes.Buffer),
		new(bytes.Buffer),
		new(bytes.Buffer),
	}
	serverWithCustomHandler := httptest.NewServer(BootStrapRouterWithCustomHandler(&MockHandlerObject))
	t.Run("Runs onConnect Function ( set by handlers )", func(t *testing.T) {
		ConnectToTestServer(t, serverWithCustomHandler)
		connectionMessage, _ := ioutil.ReadAll(MockHandlerObject.ConnectBuffer)
		assert.True(t, string(connectionMessage) == "[Connected]")
		//TODO: figure out hose to test disconnect
		//conn.Close()
		//disconnectionMessage , _ := ioutil.ReadAll(MockHandlerObject.DisconnectBuffer)
		//assert.True(t, string(disconnectionMessage) == "[Disconnected]")
	})

	var MockBuffer = MockObject{
		new(bytes.Buffer),
		new(bytes.Buffer),
		new(bytes.Buffer),
	}
	serverWithoutCustomHandler := httptest.NewServer(BootstrapRouterWithoutCustomHandler(&MockBuffer))

	t.Run("Runs onConnect Function ", func(t *testing.T) {
		ConnectToTestServer(t, serverWithoutCustomHandler)
		connectionMessage , _ := ioutil.ReadAll(MockBuffer.ConnectBuffer)
		assert.True(t, len(connectionMessage) != 0)
		//TODO: same as above
		//conn.Close()
		//disconnectionMessage , _ := ioutil.ReadAll(MockBuffer.DisconnectBuffer)
		//assert.True(t, len(disconnectionMessage) != 0)
	})


}
