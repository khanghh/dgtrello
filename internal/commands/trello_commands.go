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
	"time"

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
	members      map[string]string
	eventHub     *core.TrelloEventHub
	cancelCtx    context.CancelFunc
	mtx          sync.Mutex
}
type moduleConfig struct {
	Channels []*TrelloChannelConfig `json:"channels"`
	Members  map[string]string      `json:"members"`
}

func readConfig(configFile string) (*moduleConfig, error) {
	buf, err := ioutil.ReadFile(configFile)
	if err != nil {
		return nil, err
	}
	config := &moduleConfig{}
	if err := json.Unmarshal(buf, config); err != nil {
		return nil, err
	}
	return config, nil
}

func writeConfig(configFile string, newConfig *moduleConfig) error {
	buf, err := ioutil.ReadFile(configFile)
	if err != nil {
		return err
	}
	appConfig, err := unmarshalToMap(buf)
	if err != nil {
		return err
	}
	appConfig["channels"] = newConfig.Channels
	appConfig["members"] = newConfig.Members
	buf, err = json.MarshalIndent(appConfig, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(configFile, buf, 0644); err != nil {
		return err
	}
	log.Info("Config file saved", "channels", len(newConfig.Channels), "members", len(newConfig.Members))
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
		members:   cp.members,
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
	boardId := ctx.Arguments.Get(0).Raw()
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
	boardId := ctx.Arguments.Get(0).Raw()
	var channel *TrelloChannel
	if boardId != "" {
		channel = cp.getChannelByBoardId(boardId)
	} else {
		channel = cp.channels[ctx.Event.ChannelID]
	}
	if channel != nil {
		cp.unsubscribeTrello(channel.ChannelId())
		ctx.RespondText("OK!")
		return
	}
	ctx.RespondText("❌ Trello board not found.")
}

func (cp *TrelloCmdProcessor) memaddHandler(ctx *dgc.Ctx) {
	trelloUsername := ctx.Arguments.Get(0).Raw()
	discordUser := ctx.Arguments.Get(1).Raw()
	if userId, ok := parseUserId(discordUser); len(trelloUsername) > 0 && ok {
		cp.members[trelloUsername] = userId
		ctx.RespondText(fmt.Sprintf("Linked trello username `%s` to user <@%s>", trelloUsername, userId))
		return
	}
	ctx.RespondText("❌ Invalid arguments provided.")
}

func (cp *TrelloCmdProcessor) memdelHandler(ctx *dgc.Ctx) {
	trelloUsername := ctx.Arguments.Get(0).Raw()
	if userId, ok := cp.members[trelloUsername]; ok {
		delete(cp.members, trelloUsername)
		ctx.RespondText(fmt.Sprintf("Unlinked trello username `%s` from user <@%s>", trelloUsername, userId))
		return
	}
	ctx.RespondText("❌ Trello username not linked with any discord user.")
}

func (cp *TrelloCmdProcessor) RegisterCommands(cmdRouter *dgc.Router) {
	cmdRouter.RegisterCmd(&dgc.Command{
		Name:        "subscribe",
		Aliases:     []string{"sub"},
		Description: "Subscribe to receive events of a board on the current channel",
		Usage:       "subscribe <boardId>",
		Handler:     cp.subscribeBoardHandler,
	})
	cmdRouter.RegisterCmd(&dgc.Command{
		Name:        "unsubscribe",
		Aliases:     []string{"unsub"},
		Description: "Unsubscribe from board events of the current channel",
		Usage:       "unsubscribe [boardId]",
		Handler:     cp.unsubscribeBoardHandler,
	})
	cmdRouter.RegisterCmd(&dgc.Command{
		Name:        "memadd",
		Aliases:     []string{"memreg"},
		Description: "Add trello username to board",
		Usage:       "memadd <trello username> <discord user>",
		Handler:     cp.memaddHandler,
	})
	cmdRouter.RegisterCmd(&dgc.Command{
		Name:        "memdel",
		Description: "Remove trello username from board",
		Usage:       "memdel <trello username> <discord user>",
		Handler:     cp.memdelHandler,
	})
}

func (cp *TrelloCmdProcessor) saveConfig() {
	channels := []*TrelloChannelConfig{}
	for _, channel := range cp.channels {
		conf := TrelloChannelConfig{
			ChannelId:     channel.ChannelId(),
			BoardId:       channel.BoardId(),
			EnabledEvents: channel.listener.EnabledEvents,
			LastActionId:  channel.listener.LastActionId,
		}
		channels = append(channels, &conf)
	}
	if err := writeConfig(cp.configFile, &moduleConfig{channels, cp.members}); err != nil {
		log.Error("Could not save channels config", "error", err)
	}
}

func (cp *TrelloCmdProcessor) saveLoop(ctx context.Context) {
	for {
		select {
		case <-time.After(1 * time.Minute):
			cp.saveConfig()
		case <-ctx.Done():
			return
		}
	}
}

func (cp *TrelloCmdProcessor) OnStartBot(session *discordgo.Session) error {
	ctx, cancel := context.WithCancel(context.Background())
	cp.cancelCtx = cancel
	cp.botSession = session
	config, err := readConfig(cp.configFile)
	if err != nil {
		return err
	}
	cp.members = config.Members
	for _, conf := range config.Channels {
		if err := cp.subscribeTrello(conf); err != nil {
			log.Error(fmt.Sprintf("Failed to create trello channel. channelId: %s, boardId: %s", conf.ChannelId, conf.BoardId), "error", err)
		}
	}
	go cp.eventHub.Run(ctx)
	go cp.saveLoop(ctx)
	return nil
}

func (cp *TrelloCmdProcessor) OnStopBot() {
	cp.cancelCtx()
	cp.saveConfig()
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
