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