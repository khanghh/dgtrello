package core

import (
	"context"
	"dgtrello/internal/logger"
	"time"

	"github.com/adlio/trello"
)

type TrelloEventListener interface {
	OnTrelloEvent(action *trello.Action)
}

type TrelloEventHub struct {
	Client       *trello.Client
	pollInterval time.Duration
	listeners    map[string]TrelloEventListener
}

func (hub *TrelloEventHub) Subscribe(idModel string, listener TrelloEventListener) {
	hub.listeners[idModel] = listener
}

func (hub *TrelloEventHub) Unsubscribe(idModel string) {
	delete(hub.listeners, idModel)
}

func (hub *TrelloEventHub) pollEvents() {
	for boardId, listener := range hub.listeners {
		board := trello.Board{ID: boardId}
		board.SetClient(hub.Client)
		actions, err := board.GetActions(trello.Defaults())
		if err != nil {
			logger.Errorln("Could not get board actions. boardId:", board.ID)
			continue
		}
		for _, action := range actions {
			listener.OnTrelloEvent(action)
		}
	}
}

func (hub *TrelloEventHub) StartListening(ctx context.Context) {
	for {
		select {
		case <-time.After(hub.pollInterval):
			hub.pollEvents()
		case <-ctx.Done():
			return
		}
	}
}

func NewTrelloEventHub(client *trello.Client, pollInterval time.Duration) *TrelloEventHub {
	return &TrelloEventHub{
		Client:       client,
		pollInterval: pollInterval,
	}
}
