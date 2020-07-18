package socketeer

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"sync"
	"time"
)

var once sync.Once
type SocketeerManager struct {
	sync.Mutex
	allConnection        map[string]*websocket.Conn
	sendChannels         map[string]chan []byte
	globalActionHandlers map[string]func(message []byte, sendChannels map[string]chan []byte)
	messageHandlers      []MessageHandler
	dispatchers          []Dispatcher
	IdGen                Identifier
	Config               *Config
}

func (s *SocketeerManager) runWriter(connectionId string) {
	ticker := time.NewTicker(pingPeriod)
	connection := s.allConnection[connectionId]
	defer func() {
		ticker.Stop()
		connection.Close()
	}()

	for {
		select {
		case message, ok := <-s.sendChannels[connectionId]:
			connection.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				connection.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			err := connection.WriteMessage(websocket.TextMessage, message)

			if err != nil {
				log.Printf(fmt.Sprintf("user %s disconnected : %s \n", connectionId, err.Error()))
				return
			}
		case <-ticker.C:
			connection.SetWriteDeadline(time.Now().Add(writeWait))
			if err := connection.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (s *SocketeerManager) runReader(connectionId string) {
	for {
		connection := s.allConnection[connectionId]
		defer func() {
			connection.Close()
		}()
		connection.SetReadLimit(int64(maxMessageSize))
		connection.SetReadDeadline(time.Now().Add(pongWait))
		connection.SetPongHandler(func(string) error {
			connection.SetReadDeadline(time.Now().Add(pongWait))
			return nil
		})

		_, message, err := connection.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				//logrus.Errorf("Error occurred while reading message for %", err.Error())
				return
			}
		}

		var action *Action
		err = json.Unmarshal(message, &action)

		if err != nil {
			log.Println(err.Error())
		}

		//call custom actions
		if action != nil {
			if handler, ok := s.globalActionHandlers[action.ActionName]; ok {
				handler(message, s.sendChannels)
			}
		}

		// call MessageHandlers
		for _, handler := range s.messageHandlers {
			handler.OnMessage(message, s.sendChannels)
		}
	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func (s *SocketeerManager) Manage(response http.ResponseWriter, request *http.Request) (string, error) {
	//TODO: add the config override
	if s.allConnection == nil {
		s.Lock()
		s.allConnection = make(map[string]*websocket.Conn)
		s.Unlock()
	}

	if s.sendChannels == nil {
		s.Lock()
		s.sendChannels = make(map[string]chan []byte)
		s.Unlock()
	}

	connection, err := upgrader.Upgrade(response, request, nil)
	if err != nil {
		return "", err
	}
	id := s.IdGen.GetUniqueId()
	s.Lock()
	s.allConnection[id] = connection
	s.sendChannels[id] = make(chan []byte)
	go s.runWriter(id)
	go s.runReader(id)
	for _, dispatcher := range s.dispatchers {
		go dispatcher.Run(s)
	}
	s.Unlock()

	return id, nil
}

func (s *SocketeerManager) Broadcast(message []byte) {
	for _, channel := range s.sendChannels {
		channel <- message
	}
}

func (s *SocketeerManager) Remove(connectionId string) {
	if connection, ok := s.allConnection[connectionId]; ok {
		connection.Close()
		s.Lock()
		defer s.Unlock()
		delete(s.allConnection, connectionId)
		delete(s.sendChannels, connectionId)
	}
}

// for message handlers
func (s *SocketeerManager) AddGlobalActionHandler(actionName string, handler func(message []byte, allSendChannels map[string]chan []byte)) {
	s.Lock()
	defer s.Unlock()
	s.globalActionHandlers[actionName] = handler
}

func (s *SocketeerManager) AddMessageHandler(handler MessageHandler) {
	s.Lock()
	defer s.Unlock()
	if s.messageHandlers == nil {
		s.messageHandlers = []MessageHandler{handler}
	} else {
		s.messageHandlers = append(s.messageHandlers, handler)
	}
}

// for dispatcher handling
func (s *SocketeerManager) AddDispatcher(dispatcher Dispatcher) {
	s.Lock()
	defer s.Unlock()
	if s.dispatchers == nil {
		s.dispatchers = []Dispatcher{dispatcher}
		return
	}
	s.dispatchers = append(s.dispatchers, dispatcher)
}

func (s *SocketeerManager) SendToId(connectionId string, message []byte) {
	s.sendChannels[connectionId] <- message
}
