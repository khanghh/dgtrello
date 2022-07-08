package discordbot

import (
	"context"
	"dgtrello/internal/logger"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"time"

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
	ChannelId      string   `json:"channelId"`
	BoardId        string   `json:"boardId"`
	EnabledEvents  []string `json:"enabledEvents"`
	LastActivityId string   `json:"lastActivityId"`
}

type TrelloCmdProcessor struct {
	allowedRoles []string
	configFile   string
	channels     []*TrelloChannel
	pollInterval time.Duration
	trelloClient *trello.Client
	botSession   *discordgo.Session
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

func (tp *TrelloCmdProcessor) pollBoardEvents(boardChannel *TrelloChannel) {
	for _, channel := range tp.channels {
		board := trello.Board{ID: channel.BoardId}
		board.SetClient(tp.trelloClient)
		actions, err := board.GetActions(trello.Defaults(), trello.Defaults())
		if err != nil {
			logger.Errorln("Could not get board actions. boardId:", board.ID)
			continue
		}
		for _, action := range actions {
			fmt.Printf("%s: %s\n", action.ID, action.Type)
		}
	}
}

func (tp *TrelloCmdProcessor) startPolling(ctx context.Context) {
	for {
		select {
		case <-time.After(tp.pollInterval):
			for _, trelloChannel := range tp.channels {
				tp.pollBoardEvents(trelloChannel)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (tp *TrelloCmdProcessor) listenBoardHandler(ctx *dgc.Ctx) {
	argsBoardId := ctx.Arguments.Get(0)
	fmt.Println(argsBoardId.Raw())
	ctx.RespondText("OK")
}

func (cp *TrelloCmdProcessor) stopListenBoardHandler(ctx *dgc.Ctx) {
	ctx.RespondText("OK!")
}

func (cp *TrelloCmdProcessor) RegisterCommands(cmdRouter *dgc.Router) {
	cmdRouter.RegisterCmd(&dgc.Command{
		Name:        "trello",
		Description: "Trello command",
		Usage:       "!trello listen/stop",
		SubCommands: []*dgc.Command{
			{
				Name:        "listen",
				Description: "Listen a board",
				Usage:       "!trello listen <boardId>",
				Handler:     cp.listenBoardHandler,
			},
			{
				Name:        "stop",
				Description: "Stop listen current board",
				Usage:       "!trello stop",
				Handler:     cp.stopListenBoardHandler,
			},
		},
		Flags:   cp.allowedRoles,
		Handler: cp.listenBoardHandler,
	})
}

func (cp *TrelloCmdProcessor) OnStartBot(session *discordgo.Session) {
	ctx, cancel := context.WithCancel(context.Background())
	cp.cancelCtx = cancel

	go cp.startPolling(ctx)
}

func (cp *TrelloCmdProcessor) OnStopBot() {
	cp.cancelCtx()
}

func (cp *TrelloCmdProcessor) SetAllowedRoles(roles []string) {
	cp.allowedRoles = roles
}

func NewTrelloCommandProcessor(configFile string, trelloClient *trello.Client, pollInterval time.Duration) (*TrelloCmdProcessor, error) {
	trelloChannels, err := loadChannelConfig(configFile)
	if err != nil {
		return nil, err
	}
	return &TrelloCmdProcessor{
		configFile:   configFile,
		trelloClient: trelloClient,
		pollInterval: pollInterval,
		channels:     trelloChannels,
	}, nil
}
