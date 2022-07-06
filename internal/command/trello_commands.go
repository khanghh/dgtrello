package command

import (
	"dgtrello/internal/trello"
	"fmt"

	"github.com/lus/dgc"
)

type TrelloCmdProcessor struct {
	allowedRoles []string
	listeners    []*trello.TrelloListener
}

func (tp *TrelloCmdProcessor) testHandler(ctx *dgc.Ctx) {
	fmt.Println("guildId: ", ctx.Event.GuildID)
	guildRoles, _ := ctx.Session.GuildRoles(ctx.Event.GuildID)
	for _, role := range guildRoles {
		fmt.Printf("%s:%s\n", role.Name, role.ID)
	}
	ctx.RespondText("OK")
}

func (cp *TrelloCmdProcessor) RegisterCommands(cmdRouter *dgc.Router) {
	cmdRouter.RegisterCmd(&dgc.Command{
		Name:        "test",
		Description: "Test command",
		Usage:       "test abc def",
		Example:     "test 123 456",
		Flags:       cp.allowedRoles,
		Handler:     cp.testHandler,
	})
}

func NewTrelloCommandProcessor(roles []string) (*TrelloCmdProcessor, error) {
	return &TrelloCmdProcessor{
		allowedRoles: roles,
	}, nil
}
