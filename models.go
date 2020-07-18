package socketeer

type Dispatcher interface {
	Run(commander *SocketeerManager)
}

type MessageHandler interface {
	OnMessage(message []byte, sendChannels map[string]chan []byte)
}


type OnConnectHook func(manager *SocketeerManager , connectionId string)

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
