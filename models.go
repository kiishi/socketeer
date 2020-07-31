package socketeer

import "net/http"

type Dispatcher interface {
	Run(commander *Manager)
}
type MessageContext struct{
	From string
	Body []byte
}

type MessageHandler interface {
	OnMessage( manager *Manager , ctx *MessageContext )
}

type IdGen func() string

type OnDisconnectFunc func(manager *Manager , connectionId string)

type OnConnectFunc func(manager *Manager , request *http.Request , connectionId string)

type Identifier interface {
	GetUniqueId() string
}

type Action struct {
	ActionName string `json:"action"`
}
type ActionHandler func(message []byte, allSendChannels map[string]chan []byte)

type Config struct {
	PongWait           int
	PingPeriod         int
	WriteWait          int
	MaxMessageSize     int64
	MaxReadBufferSize  int
	MaxWriteBufferSize int
}
