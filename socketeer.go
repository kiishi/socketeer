package socketeer

import (
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"sync"
	"time"
)

func NewManager() *Manager {
	return &Manager{}
}

type Manager struct {
	sync.Mutex
	initialized     bool
	allConnection   map[string]*websocket.Conn
	sendChannels    map[string]chan []byte
	messageHandlers []MessageHandler
	dispatchers     []Dispatcher
	onConnect       OnConnectFunc
	onDisconnect    OnDisconnectFunc
	IdGen           IdFactory
	Config          *Config
}

func (s *Manager) OnConnect(onConnectHandler OnConnectFunc) {
	s.onConnect = onConnectHandler
}


func (s *Manager) OnDisconnect(onDisconnectHandler OnDisconnectFunc) {
	s.onDisconnect = onDisconnectHandler
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
	connection := s.allConnection[connectionId]
	defer func() {
		connection.Close()
	}()

	for {
		connection.SetReadLimit(maxMessageSize)
		connection.SetReadDeadline(time.Now().Add(pongWait))
		connection.SetPongHandler(func(string) error {
			connection.SetReadDeadline(time.Now().Add(pongWait))
			return nil
		})

		_, message, err := connection.ReadMessage()

		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) || websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				s.Lock()
				delete(s.allConnection, connectionId)
				delete(s.sendChannels, connectionId)
				s.Unlock()
				if s.onDisconnect != nil {
					s.onDisconnect(s, connectionId)
				}
				// if all handlers have an onDisconnectFunction
				for _ , handler := range s.messageHandlers{
					if instanceDisconnectFunc , ok := handler.(OnDisconnectHandler); ok{
						instanceDisconnectFunc.OnDisconnect(s , connectionId)
					}
				}
				return
			} else {
				fmt.Printf("Socketeer Error for connection %s ==> %s \n", connectionId, err.Error())
				return
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
	s.Unlock()

	if s.onConnect != nil {
		//d
		go s.onConnect(s, request, id)
	}

	for _ , handler := range s.messageHandlers{
		if instanceOnConnectFunc ,ok := handler.(OnConnectHandler);ok{
			instanceOnConnectFunc.OnConnect(s, request , id)
		}
	}
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

func (s *Manager) AddIdFactory(idGen IdFactory) {
	s.IdGen = idGen
}
