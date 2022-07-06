package discordbot

import (
	"github.com/bwmarrin/discordgo"
	"github.com/lus/dgc"
)

const defaultCmdPrefix = "!"

type CommandProcessor interface {
	RegisterCommands(cmdRouter *dgc.Router)
}

type DiscordBot struct {
	Session   *discordgo.Session
	CmdRouter *dgc.Router
}

func (bot *DiscordBot) RegisterCommand(cmds ...*dgc.Command) {
	for _, cmd := range cmds {
		bot.CmdRouter.RegisterCmd(cmd)
	}
}

func (bot *DiscordBot) AddProcessor(processor CommandProcessor) {
	processor.RegisterCommands(bot.CmdRouter)
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

func (bot *DiscordBot) Run() {
	bot.CmdRouter.RegisterMiddleware(restrictRolesMiddleware)
	bot.CmdRouter.RegisterDefaultHelpCommand(bot.Session, nil)
	bot.CmdRouter.Initialize(bot.Session)
}
