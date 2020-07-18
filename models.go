package socketeer

type Dispatcher interface {
	Run(commander *SocketeerManager)
}

type MessageHandler interface {
	OnMessage(message []byte, sendChannels map[string]chan []byte)
}

type Identifier interface {
	GetUniqueId() string
}

type Action struct {
	ActionName string `json:"action"`
}

type Config struct {
	PongWait           uint32
	PingPeriod         uint32
	WriteWait          uint32
	MaxMessageSize     uint32
	MaxReadBufferSize  uint32
	MaxWriteBufferSize uint32
}
