package discordbot

import (
	"github.com/bwmarrin/discordgo"
	"github.com/lus/dgc"
)

const cmdPrefix = "!"

type DiscordBot struct {
	session   *discordgo.Session
	cmdRouter *dgc.Router
}

func onlyRoles(next dgc.ExecutionHandler, roles []string) dgc.ExecutionHandler {
	return func(ctx *dgc.Ctx) {
		if ctx.Event.GuildID == "" {
			ctx.RespondText("You do not have permission to perform this action.")
			return
		}
		guildRoles, _ := ctx.Session.GuildRoles(ctx.Event.GuildID)
		for _, role := range guildRoles {
			for _, roleId := range ctx.Event.Member.Roles {
				if roleId == role.ID {
					next(ctx)
					return
				}
			}
		}
	}
}

func (bot *DiscordBot) RegisterCmd(cmd *dgc.Command, roles []string) {
	if len(roles) > 0 {
		cmd.Handler = onlyRoles(cmd.Handler, roles)
	}
	bot.cmdRouter.RegisterCmd(cmd)
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
	cmdRouter := dgc.Create(&dgc.Router{
		Prefixes: []string{cmdPrefix},
	})
	return &DiscordBot{
		session:   botSession,
		cmdRouter: cmdRouter,
	}, nil
}

func (bot *DiscordBot) Run() {
	bot.cmdRouter.RegisterDefaultHelpCommand(bot.session, nil)
	bot.cmdRouter.Initialize(bot.session)
}
