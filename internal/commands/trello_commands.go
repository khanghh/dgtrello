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

type TrelloChannel struct {
	botSession    *discordgo.Session
	ChannelId     string   `json:"channelId"`
	BoardId       string   `json:"boardId"`
	EnabledEvents []string `json:"enabledEvents"`
	LastActionId  string   `json:"lastActionId"`
}

func (ch *TrelloChannel) OnTrelloEvent(action *trello.Action) {
	fmt.Println(action.Type)
}

type TrelloCmdProcessor struct {
	botSession   *discordgo.Session
	allowedRoles []string
	configFile   string
	channels     map[string]*TrelloChannel
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

func (cp *TrelloCmdProcessor) subscribeTrello(channel *TrelloChannel) {
	channel.botSession = cp.botSession
	cp.channels[channel.ChannelId] = channel

	cp.eventHub.Subscribe(channel.BoardId, channel)
}

func (cp *TrelloCmdProcessor) unsubscribeTrello(channelId string) {
	channel := cp.channels[channelId]
	cp.eventHub.Unsubscribe(channel.BoardId)
	delete(cp.channels, channelId)
}

func (cp *TrelloCmdProcessor) getChannelByBoardId(boardId string) *TrelloChannel {
	for _, channel := range cp.channels {
		if channel.BoardId == boardId {
			return channel
		}
	}
	return nil
}

func (cp *TrelloCmdProcessor) watchBoardHandler(ctx *dgc.Ctx) {
	argBoardId := ctx.Arguments.Get(0)
	boardId := argBoardId.Raw()
	_, err := cp.eventHub.Client.GetBoard(boardId, trello.Defaults())
	if err != nil {
		ctx.RespondText(fmt.Sprintf("Could not find board %s", boardId))
		return
	}
	if channel := cp.getChannelByBoardId(boardId); channel != nil {
		ctx.RespondText(fmt.Sprintf("Already watching board %s", boardId))
		return
	}
	cp.subscribeTrello(&TrelloChannel{
		ChannelId: ctx.Event.ChannelID,
		BoardId:   boardId,
	})
	ctx.RespondText("OK")
}

func (cp *TrelloCmdProcessor) stopWatchBoardHandler(ctx *dgc.Ctx) {
	if channel, ok := cp.channels[ctx.Event.ChannelID]; ok {
		cp.unsubscribeTrello(channel.ChannelId)
		ctx.RespondText("OK!")
		return
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
		Usage: "trello stop/watch",
	})
}

func (cp *TrelloCmdProcessor) OnStartBot(session *discordgo.Session) error {
	ctx, cancel := context.WithCancel(context.Background())
	cp.cancelCtx = cancel
	cp.botSession = session
	trelloChannels, err := loadChannelConfig(cp.configFile)
	if err != nil {
		return err
	}
	for _, channel := range trelloChannels {
		cp.subscribeTrello(channel)
	}
	go cp.eventHub.StartListening(ctx)
	return nil
}

func (cp *TrelloCmdProcessor) OnStopBot() {
	cp.cancelCtx()
}

func (cp *TrelloCmdProcessor) SetAllowedRoles(roles []string) {
	cp.allowedRoles = roles
}

func NewTrelloCommandProcessor(configFile string, trelloEventHub *core.TrelloEventHub) (*TrelloCmdProcessor, error) {
	return &TrelloCmdProcessor{
		configFile: configFile,
		eventHub:   trelloEventHub,
		channels:   map[string]*TrelloChannel{},
	}, nil
}
