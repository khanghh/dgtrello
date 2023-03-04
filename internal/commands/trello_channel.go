package commands

import (
	"dgtrello/internal/core"
	"fmt"

	"github.com/adlio/trello"
	"github.com/bwmarrin/discordgo"
)

type TrelloChannelConfig struct {
	ChannelId     string   `json:"channelId"`
	BoardId       string   `json:"boardId"`
	EnabledEvents []string `json:"enabledEvents"`
	LastActionId  string   `json:"lastActionId"`
}

type TrelloChannel struct {
	channelId string
	session   *discordgo.Session
	listener  *core.TrelloEventListener
}

func (ch *TrelloChannel) BoardId() string {
	return ch.listener.IdModel
}

func (ch *TrelloChannel) ChannelId() string {
	return ch.channelId
}

func (ch *TrelloChannel) OnTrelloEvent(ctx *core.TrelloEventCtx, action *trello.Action) {
	fmt.Printf("%s: %s\n", action.Type, action.ID)
}
