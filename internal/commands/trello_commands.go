package commands

import (
	"context"
	"dgtrello/internal/core"
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/adlio/trello"
	"github.com/bwmarrin/discordgo"
	"github.com/lus/dgc"
)

type TrelloBoardEvent string

const (
	EventCreateCard  = "createCard"
	EventCopyCard    = "copyCard"
	EventCommentCard = "commentCard"
	EventDeleteCard  = "deleteCard"
	EventUpdateCard  = "updateCard"
)

type TrelloChannel struct {
	botSession    *discordgo.Session
	ChannelId     string   `json:"channelId"`
	BoardId       string   `json:"boardId"`
	EnabledEvents []string `json:"enabledEvents"`
	LastActionId  string   `json:"lastActionId"`
}

func (ch *TrelloChannel) OnTrelloEvent(action *trello.Action) {
}

type TrelloCmdProcessor struct {
	allowedRoles []string
	configFile   string
	channels     []*TrelloChannel
	eventHub     *core.TrelloEventHub
	cancelCtx    context.CancelFunc
}

func loadChannelConfig(configFile string) ([]*TrelloChannel, error) {
	type moduleConfig struct {
		Channels []*TrelloChannel `json:"channels"`
	}
	buf, err := ioutil.ReadFile(configFile)
	if err != nil {
		return nil, err
	}
	configData := &moduleConfig{}
	if err := json.Unmarshal(buf, configData); err != nil {
		return nil, err
	}
	return configData.Channels, nil
}

func (tp *TrelloCmdProcessor) watchBoardHandler(ctx *dgc.Ctx) {
	argBoardId := ctx.Arguments.Get(0)
	boardId := argBoardId.Raw()
	_, err := tp.eventHub.Client.GetBoard(boardId, trello.Defaults())
	if err != nil {
		ctx.RespondText(fmt.Sprintf("Could not find board %s", boardId))
		return
	}
	trelloChannel := &TrelloChannel{
		botSession: ctx.Session,
		ChannelId:  ctx.Event.ChannelID,
		BoardId:    boardId,
	}
	tp.eventHub.Subscribe(boardId, trelloChannel)
	ctx.RespondText("OK")
}

func (cp *TrelloCmdProcessor) stopWatchBoardHandler(ctx *dgc.Ctx) {
	channelId := ctx.Event.ChannelID
	for _, channel := range cp.channels {
		if channel.ChannelId == channelId {
			cp.eventHub.Unsubscribe(channel.BoardId)
			ctx.RespondText("OK!")
			return
		}
	}
	ctx.RespondText("No board is watching")
}

func (cp *TrelloCmdProcessor) RegisterCommands(cmdRouter *dgc.Router) {
	cmdRouter.RegisterCmd(&dgc.Command{
		Name:        "trello",
		Description: "Bot commands for trello",
		SubCommands: []*dgc.Command{
			{
				Name:        "watch",
				Description: "Watch a board",
				Usage:       "trello watch <boardId>",
				Handler:     cp.watchBoardHandler,
			},
			{
				Name:        "stop",
				Description: "Stop watching current board",
				Usage:       "trello stop",
				Handler:     cp.stopWatchBoardHandler,
			},
		},
		Flags: cp.allowedRoles,
	})
}

func (cp *TrelloCmdProcessor) OnStartBot(session *discordgo.Session) {
	ctx, cancel := context.WithCancel(context.Background())
	cp.cancelCtx = cancel
	go cp.eventHub.StartListening(ctx)
}

func (cp *TrelloCmdProcessor) OnStopBot() {
	cp.cancelCtx()
}

func (cp *TrelloCmdProcessor) SetAllowedRoles(roles []string) {
	cp.allowedRoles = roles
}

func NewTrelloCommandProcessor(configFile string, trelloEventHub *core.TrelloEventHub) (*TrelloCmdProcessor, error) {
	trelloChannels, err := loadChannelConfig(configFile)
	if err != nil {
		return nil, err
	}
	return &TrelloCmdProcessor{
		configFile: configFile,
		eventHub:   trelloEventHub,
		channels:   trelloChannels,
	}, nil
}
