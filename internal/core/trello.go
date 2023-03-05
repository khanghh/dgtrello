package core

import (
	"context"
	"errors"
	"time"

	log "github.com/inconshreveable/log15"

	"github.com/adlio/trello"
)

const (
	EventCreateCard  = "createCard"
	EventCopyCard    = "copyCard"
	EventCommentCard = "commentCard"
	EventDeleteCard  = "deleteCard"
	EventUpdateCard  = "updateCard"
)

var (
	ErrAlreadySubscribe = errors.New("already subscribe")
	ErrNoEventListener  = errors.New("event listener not found")
)

type TrelloEventHandler func(ctx *TrelloEventCtx, action *trello.Action)

type TrelloEventCtx struct {
	IdModel       string
	EnabledEvents []string
	LastActionId  string
}

type TrelloEventListener struct {
	*TrelloEventCtx
	Handler TrelloEventHandler
}

type TrelloEventHub struct {
	Client       *trello.Client
	pollInterval time.Duration
	listeners    map[string]*TrelloEventListener
}

func (hub *TrelloEventHub) Listeners() []*TrelloEventListener {
	ret := make([]*TrelloEventListener, 0)
	for _, listener := range hub.listeners {
		ret = append(ret, listener)
	}
	return ret
}

func (hub *TrelloEventHub) GetListener(idModel string) *TrelloEventListener {
	return hub.listeners[idModel]
}

func (hub *TrelloEventHub) Subscribe(idModel string, events []string, lastActionId string, handler TrelloEventHandler) (*TrelloEventListener, error) {
	if listener, exist := hub.listeners[idModel]; exist {
		return listener, ErrAlreadySubscribe
	}
	hub.listeners[idModel] = &TrelloEventListener{
		TrelloEventCtx: &TrelloEventCtx{
			IdModel:       idModel,
			EnabledEvents: events,
			LastActionId:  lastActionId,
		},
		Handler: handler,
	}
	return hub.listeners[idModel], nil
}

func (hub *TrelloEventHub) Unsubscribe(idModel string) {
	delete(hub.listeners, idModel)
}

func arrayContains(arr []string, str string) bool {
	for _, item := range arr {
		if item == str {
			return true
		}
	}
	return false
}

func (hub *TrelloEventHub) pollEvents() {
	for boardId, listener := range hub.listeners {
		board := trello.Board{ID: boardId}
		board.SetClient(hub.Client)
		actions, err := board.GetActions(trello.Defaults())
		if err != nil {
			log.Error("Could not fetch board events", "boardId", board.ID, "err", err)
			continue
		}
		for _, action := range actions {
			if action.ID > listener.LastActionId && arrayContains(listener.EnabledEvents, action.Type) {
				if listener.Handler != nil {
					listener.Handler(listener.TrelloEventCtx, action)
					listener.TrelloEventCtx.LastActionId = action.ID
				}
			}
		}
	}
}

func (hub *TrelloEventHub) Run(ctx context.Context) {
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
		listeners:    map[string]*TrelloEventListener{},
	}
}
