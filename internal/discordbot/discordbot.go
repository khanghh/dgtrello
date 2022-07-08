package discordbot

import (
	"context"

	"github.com/bwmarrin/discordgo"
	"github.com/lus/dgc"
)

const defaultCmdPrefix = "!"

type CommandProcessor interface {
	RegisterCommands(cmdRouter *dgc.Router)
	OnStartBot(session *discordgo.Session)
	OnStopBot()
}

type Ctx struct {
}

type DiscordBot struct {
	Session       *discordgo.Session
	CmdRouter     *dgc.Router
	cmdProcessors []CommandProcessor
}

func (bot *DiscordBot) RegisterCommand(cmds ...*dgc.Command) {
	for _, cmd := range cmds {
		bot.CmdRouter.RegisterCmd(cmd)
	}
}

func (bot *DiscordBot) AddCommandProcessor(processor CommandProcessor) {
	bot.cmdProcessors = append(bot.cmdProcessors, processor)
}

func (bot *DiscordBot) SetCmdPrefix(cmdPrefix string) {
	bot.CmdRouter.Prefixes = []string{cmdPrefix}
}

func NewDiscordBot(botToken string) (*DiscordBot, error) {
	botSession, err := discordgo.New("Bot " + botToken)
	if err != nil {
		return nil, err
	}
	err = botSession.Open()
	if err != nil {
		return nil, err
	}
	cmdRouter := &dgc.Router{
		Prefixes: []string{defaultCmdPrefix},
		Storage:  make(map[string]*dgc.ObjectsMap),
	}
	return &DiscordBot{
		Session:   botSession,
		CmdRouter: cmdRouter,
	}, nil
}

func (bot *DiscordBot) Run(ctx context.Context) {
	bot.CmdRouter.RegisterMiddleware(restrictRolesMiddleware)
	for _, processor := range bot.cmdProcessors {
		processor.RegisterCommands(bot.CmdRouter)
	}
	bot.CmdRouter.RegisterDefaultHelpCommand(bot.Session, nil)
	bot.CmdRouter.Initialize(bot.Session)

	for _, processor := range bot.cmdProcessors {
		processor.OnStartBot(bot.Session)
	}
	defer func() {
		<-ctx.Done()
		for _, processor := range bot.cmdProcessors {
			processor.OnStopBot()
		}
	}()
}
