package commands

import (
	"context"
	"dgtrello/internal/core"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"sync"

	"github.com/adlio/trello"
	"github.com/bwmarrin/discordgo"
	log "github.com/inconshreveable/log15"
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

func unmarshalToMap(data []byte) (map[string]interface{}, error) {
	ret := make(map[string]interface{})
	if err := json.Unmarshal(data, &ret); err != nil {
		return nil, err
	}
	return ret, nil
}

func saveChannelConfig(configFile string, channels []*TrelloChannelConfig) error {
	buf, err := ioutil.ReadFile(configFile)
	if err != nil {
		return err
	}
	config, err := unmarshalToMap(buf)
	if err != nil {
		return err
	}
	config["channels"] = channels
	buf, err = json.MarshalIndent(config, "", " ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(configFile, buf, 0644); err != nil {
		return err
	}
	log.Info("Saved channels config", "count", len(channels))
	return nil
}

func (cp *TrelloCmdProcessor) subscribeTrello(conf *TrelloChannelConfig) error {
	cp.mtx.Lock()
	defer cp.mtx.Unlock()
	if _, exist := cp.channels[conf.ChannelId]; exist {
		return errAlreadyBind
	}
	channel := &TrelloChannel{
		channelId: conf.ChannelId,
		session:   cp.botSession,
	}
	listener, err := cp.eventHub.Subscribe(conf.BoardId, conf.EnabledEvents, conf.LastActionId, channel.OnTrelloEvent)
	if err != nil {
		return err
	}
	log.Info(fmt.Sprintf("Subscribed Trello boardId: `%s`, channelId: %s, events: [%s]", conf.BoardId, conf.ChannelId, strings.Join(conf.EnabledEvents, ",")))
	channel.listener = listener
	cp.channels[conf.ChannelId] = channel
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
	log.Info(fmt.Sprintf("Unsubscribed Trello boardId: `%s`, channelId: %s", channel.BoardId(), channel.ChannelId()))
}

func (cp *TrelloCmdProcessor) getChannelByBoardId(boardId string) *TrelloChannel {
	for _, channel := range cp.channels {
		if channel.BoardId() == boardId {
			return channel
		}
	}
	return nil
}

func (cp *TrelloCmdProcessor) subscribeBoardHandler(ctx *dgc.Ctx) {
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
	conf := &TrelloChannelConfig{
		ChannelId: ctx.Event.ChannelID,
		BoardId:   boardId,
		EnabledEvents: []string{
			core.EventCreateCard,
			core.EventCopyCard,
			core.EventCommentCard,
			core.EventDeleteCard,
			core.EventUpdateCard,
		},
	}
	if err := cp.subscribeTrello(conf); err != nil {
		log.Error(fmt.Sprintf("Could not subscribe board %s", boardId), "channelId", conf.ChannelId, "error", err)
		ctx.RespondText(fmt.Sprintf("Failed to subscribe board events, see log for more detail. (boardId: %s)", boardId))
		return
	}
	ctx.RespondText(fmt.Sprintf("Subscribed Trello board `%s` and notify to this channel", boardId))
}

func (cp *TrelloCmdProcessor) unsubscribeBoardHandler(ctx *dgc.Ctx) {
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
				Name:        "subscribe",
				Aliases:     []string{"watch"},
				Description: "Subscribe to receive events of a board on the current channel",
				Usage:       "trello subscribe <boardId>",
				Handler:     cp.subscribeBoardHandler,
			},
			{
				Name:        "unsubscribe",
				Aliases:     []string{"unwatch"},
				Description: "Unsubscribe from board events of the current channel",
				Usage:       "trello unsubscribe",
				Handler:     cp.unsubscribeBoardHandler,
			},
		},
		Flags: cp.allowedRoles,
		Usage: "trello [subscribe|unsubscribe]",
		Handler: func(ctx *dgc.Ctx) {
			ctx.RespondText("Unknown command")
		},
	})
}

func (cp *TrelloCmdProcessor) OnStartBot(session *discordgo.Session) error {
	ctx, cancel := context.WithCancel(context.Background())
	cp.cancelCtx = cancel
	cp.botSession = session
	channelsCgf, err := loadChannelConfig(cp.configFile)
	if err != nil {
		return err
	}
	for _, conf := range channelsCgf {
		if err := cp.subscribeTrello(conf); err != nil {
			log.Error(fmt.Sprintf("Failed to create trello channel. channelId: %s, boardId: %s", conf.ChannelId, conf.BoardId), "error", err)
		}
	}
	go cp.eventHub.Run(ctx)
	return nil
}

func (cp *TrelloCmdProcessor) OnStopBot() {
	cp.cancelCtx()
	channlesCfg := []*TrelloChannelConfig{}
	for _, channel := range cp.channels {
		conf := TrelloChannelConfig{
			ChannelId:     channel.ChannelId(),
			BoardId:       channel.BoardId(),
			EnabledEvents: channel.listener.EnabledEvents,
			LastActionId:  channel.listener.LastActionId,
		}
		channlesCfg = append(channlesCfg, &conf)
	}
	if err := saveChannelConfig(cp.configFile, channlesCfg); err != nil {
		log.Error("Could not save channels config", "error", err)
	}
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
