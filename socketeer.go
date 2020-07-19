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

type Manager struct {
	sync.Mutex
	initialized          bool
	allConnection        map[string]*websocket.Conn
	sendChannels         map[string]chan []byte
	globalActionHandlers map[string]func(message []byte, sendChannels map[string]chan []byte)
	messageHandlers      []MessageHandler
	dispatchers          []Dispatcher
	OnConnect      OnConnectHook
	onDisconnectHooks    []OnDisconnectHook
	IdGen                IdGen
	Config               *Config
}

func (s *Manager) Init() {
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

	if s.Config != nil {
		if s.Config.MaxMessageSize != 0 {
			maxMessageSize = s.Config.MaxMessageSize
		}

		if s.Config.PongWait != 0 {
			pongWait = time.Duration(s.Config.PongWait) * time.Second
		}

		if s.Config.MaxMessageSize != 0 {
			maxMessageSize = s.Config.MaxMessageSize
		}

		if s.Config.MaxReadBufferSize != 0 {
			maxReadBufferSize = s.Config.MaxReadBufferSize
		}

		if s.Config.MaxWriteBufferSize != 0 {
			maxWriteBufferSize = s.Config.MaxWriteBufferSize
		}

		if s.Config.WriteWait != 0 {
			writeWait = time.Duration(s.Config.WriteWait) * time.Second
		}
	}

	for _, dispatcher := range s.dispatchers {
		go dispatcher.Run(s)
	}

	s.initialized = true
}


func (s *Manager) runWriter(connectionId string) {
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

func (s *Manager) runReader(connectionId string) {
	for {
		connection := s.allConnection[connectionId]
		defer func() {
			connection.Close()
		}()

		connection.SetReadLimit(maxMessageSize)
		connection.SetReadDeadline(time.Now().Add(pongWait))
		connection.SetPongHandler(func(string) error {
			connection.SetReadDeadline(time.Now().Add(pongWait))
			return nil
		})

		_ , message, err := connection.ReadMessage()

		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				s.Lock()
				delete(s.allConnection, connectionId)
				delete(s.sendChannels, connectionId)
				s.Unlock()

				for _, hook := range s.onDisconnectHooks {
					go hook(s, connectionId)
				}
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
			handler.OnMessage(s, &MessageContext{
				From: connectionId,
				Body: message,
			})
		}
	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	ReadBufferSize:  maxReadBufferSize,
	WriteBufferSize: maxWriteBufferSize,
}

func (s *Manager) Manage(response http.ResponseWriter, request *http.Request) (string, error) {
	connection, err := upgrader.Upgrade(response, request, nil)
	if s.initialized == false {
		panic("Socketeer not Initialized, Call Init()")
	}
	if err != nil {
		return "", err
	}
	id := s.IdGen()
	s.Lock()
	s.allConnection[id] = connection
	s.sendChannels[id] = make(chan []byte)
	go s.runWriter(id)
	go s.runReader(id)
	go s.OnConnect(s, request, id)
	s.Unlock()
	return id, nil
}

func (s *Manager) Broadcast(message []byte) {
	for _, channel := range s.sendChannels {
		channel <- message
	}
}

func (s *Manager) Remove(connectionId string) {
	if connection, ok := s.allConnection[connectionId]; ok {
		connection.Close()
		s.Lock()
		defer s.Unlock()
		delete(s.allConnection, connectionId)
		delete(s.sendChannels, connectionId)
	}
}

// for message handlers
func (s *Manager) AddGlobalActionHandler(actionName string, handler ActionHandler) {
	s.Lock()
	defer s.Unlock()
	s.globalActionHandlers[actionName] = handler
}

func (s *Manager) AddMessageHandler(handler MessageHandler) {
	s.Lock()
	defer s.Unlock()
	if s.messageHandlers == nil {
		s.messageHandlers = []MessageHandler{handler}
	} else {
		s.messageHandlers = append(s.messageHandlers, handler)
	}
}

// for dispatcher handling
func (s *Manager) AddDispatcher(dispatcher Dispatcher) {
	s.Lock()
	defer s.Unlock()
	if s.dispatchers == nil {
		s.dispatchers = []Dispatcher{dispatcher}
		return
	}
	s.dispatchers = append(s.dispatchers, dispatcher)
}

func (s *Manager) SendToId(connectionId string, message []byte) error {
	if user, ok := s.sendChannels[connectionId]; ok {
		user <- message
		return nil
	} else {
		return ConnectionIdDoestExist
	}

}


func ( s *Manager ) AddIdFactory(idGen IdGen) {
	s.IdGen = idGen
}
