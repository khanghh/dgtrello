package discordbot

import (
	"github.com/bwmarrin/discordgo"
	"github.com/lus/dgc"
)

const cmdPrefix = "!"

type CommandProcessor interface {
	RegisterCommands(cmdRouter *dgc.Router)
}

type DiscordBot struct {
	Session   *discordgo.Session
	CmdRouter *dgc.Router
}

func restrictRolesMiddleware(next dgc.ExecutionHandler) dgc.ExecutionHandler {
	return func(ctx *dgc.Ctx) {
		for _, allowedRoleId := range ctx.Command.Flags {
			for _, roleId := range ctx.Event.Member.Roles {
				if roleId == allowedRoleId {
					next(ctx)
					return
				}
			}
		}
		ctx.RespondText("You do not have permission to perform this action.")
	}
}

func (bot *DiscordBot) RegisterCommand(cmds ...*dgc.Command) {
	for _, cmd := range cmds {
		bot.CmdRouter.RegisterCmd(cmd)
	}
}

func (bot *DiscordBot) AddProcessor(processor CommandProcessor) {
	processor.RegisterCommands(bot.CmdRouter)
}

func NewDiscordBot(botToken string, channelId string) (*DiscordBot, error) {
	botSession, err := discordgo.New("Bot " + botToken)
	if err != nil {
		return nil, err
	}
	err = botSession.Open()
	if err != nil {
		return nil, err
	}
	cmdRouter := &dgc.Router{
		Prefixes: []string{cmdPrefix},
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
