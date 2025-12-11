package worker

import (
	"log"

	"github.com/johndosdos/chatter/internal/model"
	"github.com/johndosdos/chatter/internal/websocket"
)

func WorkerHub(hub *websocket.Hub) func(model.Message) {
	return func(payload model.Message) {
		log.Println("payload received")
		hub.FromWorker <- payload
	}
}
