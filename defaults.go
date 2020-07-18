package socketeer

import "time"

var (
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	writeWait      = 10 * time.Second
	maxMessageSize = uint32(512)
	maxReadBufferSize = 1024
	maxWriteBufferSize = 1024
)