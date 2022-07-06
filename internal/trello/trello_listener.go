package trello

import (
	trelloapi "github.com/adlio/trello"
)

const defaultPollInterval = 1000

type TrelloListener struct {
	ApiKey        string
	AuthToken     string
	BoardIDs      []string
	EnabledEvents []string
	PollInterval  int
	client        *trelloapi.Client
}

func (tl *TrelloListener) StartListening() {
	tl.client = trelloapi.NewClient(tl.ApiKey, tl.AuthToken)
}
