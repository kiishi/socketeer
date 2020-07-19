package main

import (
	"encoding/json"
	"github.com/kiishi/socketeer"
	"log"
	"sync"
)

type ChatStoreObject struct {
	sync.Mutex
	AllUsers map[string]interface{}
	Rooms    map[string][]string
}

type Message struct {
	Id          string `json:"id"`
	Type        string `json:"type"`
	From        string `json:"from"`
	To          string `json:"to"`
	Room        string `json:"room"`
	MessageBody string `json:"message_body"`
}

func (c *ChatStoreObject) OnMessage(manager *socketeer.Manager, ctx *socketeer.MessageContext) {
	var message *Message
	err := json.Unmarshal(ctx.Body, &message)

	if err != nil {
		log.Printf("Revceived badMessage format from %s", ctx.From)
		return
	}

	switch message.Type {
	case "room":
		if room, ok := c.Rooms[message.Room]; ok {
			for _, user := range room {
				err := manager.SendToId(user, ctx.Body)
				if err != nil {
					log.Println("User not found")
				}
			}
		}
	case "private":
		err := manager.SendToId(message.To, ctx.Body)
		if err != nil {
			log.Println("User not found")
		}

	case "join":
		if _, ok := c.Rooms[message.Room]; ok {
			c.Lock()
			c.Rooms[message.Room] = append(c.Rooms[message.Room], ctx.From)
			//	You can send an acknowledgement message here
		} else {
			//	Send a message back if room doesnt exit
		}
	default:
		ErrorMessage, err := json.Marshal(&struct {
			Error   bool   `json:"error"`
			Message string `json:"message"`
		}{
			true, "Invalid action",
		})

		if err != nil {
			log.Println("Cannot parse error message")
			return
		}
		manager.SendToId(ctx.From, ErrorMessage)
	}
}
