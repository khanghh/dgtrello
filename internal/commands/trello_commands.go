package commands

import (
	"context"
	"dgtrello/internal/core"
	"dgtrello/internal/logger"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"sync"

	"github.com/adlio/trello"
	"github.com/bwmarrin/discordgo"
	"github.com/lus/dgc"
)

var (
	errAlreadyBind = errors.New("already bind")
)

type TrelloCmdProcessor struct {
	botSession   *discordgo.Session
	allowedRoles []string
	configFile   string
	channels     map[string]*TrelloChannel
	eventHub     *core.TrelloEventHub
	cancelCtx    context.CancelFunc
	mtx          sync.Mutex
}

func loadChannelConfig(configFile string) ([]*TrelloChannelConfig, error) {
	type moduleConfig struct {
		Channels []*TrelloChannelConfig `json:"channels"`
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

func (cp *TrelloCmdProcessor) subscribeTrello(cfg *TrelloChannelConfig) error {
	cp.mtx.Lock()
	defer cp.mtx.Unlock()
	if _, exist := cp.channels[cfg.ChannelId]; exist {
		return errAlreadyBind
	}
	channel := &TrelloChannel{
		channelId: cfg.ChannelId,
		session:   cp.botSession,
	}
	listener, err := cp.eventHub.Subscribe(cfg.BoardId, cfg.EnabledEvents, cfg.LastActionId, channel.OnTrelloEvent)
	if err != nil {
		return err
	}
	channel.listener = listener
	cp.channels[cfg.ChannelId] = channel
	return nil
}

func (cp *TrelloCmdProcessor) unsubscribeTrello(channelId string) {
	cp.mtx.Lock()
	defer cp.mtx.Unlock()
	channel, exist := cp.channels[channelId]
	if !exist {
		return
	}
	cp.eventHub.Unsubscribe(channel.BoardId())
	delete(cp.channels, channelId)
}

func (cp *TrelloCmdProcessor) getChannelByBoardId(boardId string) *TrelloChannel {
	for _, channel := range cp.channels {
		if channel.BoardId() == boardId {
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
	listener := cp.eventHub.GetListener(boardId)
	if listener != nil {
		ctx.RespondText(fmt.Sprintf("Already watching board %s", boardId))
		return
	}
	channelCfg := &TrelloChannelConfig{
		BoardId: boardId,
		EnabledEvents: []string{
			core.EventCreateCard,
			core.EventCopyCard,
			core.EventCommentCard,
			core.EventDeleteCard,
			core.EventUpdateCard,
		},
		LastActionId: "",
	}
	if err := cp.subscribeTrello(channelCfg); err != nil {
		ctx.RespondText(fmt.Sprintf("Failed to watch board events, see log for more detail. (boardId: %s)", boardId))
		return
	}
}

func (cp *TrelloCmdProcessor) stopWatchBoardHandler(ctx *dgc.Ctx) {
	if channel, ok := cp.channels[ctx.Event.ChannelID]; ok {
		cp.unsubscribeTrello(channel.ChannelId())
		ctx.RespondText("OK!")
		return
	}
	ctx.RespondText("Not watching any board")
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
	channelConfigs, err := loadChannelConfig(cp.configFile)
	if err != nil {
		return err
	}
	for _, conf := range channelConfigs {
		if err := cp.subscribeTrello(conf); err != nil {
			logger.Errorln(fmt.Sprintf("Failed to create trello channel. channelId: %s, boardId: %s", conf.ChannelId, conf.BoardId))
			logger.Errorln(err)
		}
		logger.Printf("Listening events for board: %s, channel: %s", conf.BoardId, conf.ChannelId)
	}
	go cp.eventHub.Run(ctx)
	return nil
}

func (cp *TrelloCmdProcessor) OnStopBot() {
	cp.cancelCtx()
}

func (cp *TrelloCmdProcessor) SetAllowedRoles(roles []string) {
	cp.allowedRoles = roles
}

func NewTrelloCommandProcessor(channelCfg string, trelloEventHub *core.TrelloEventHub) (*TrelloCmdProcessor, error) {
	return &TrelloCmdProcessor{
		configFile: channelCfg,
		eventHub:   trelloEventHub,
		channels:   make(map[string]*TrelloChannel),
	}, nil
}
