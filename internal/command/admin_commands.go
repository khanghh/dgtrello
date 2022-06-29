package discordbot

import (
	"github.com/lus/dgc"
)

type adminCmdProcessor struct {
	adminRoles []string
}

func (ac *adminCmdProcessor) RegisterCommands(cmdRouter *dgc.Router) {
	cmdRouter.RegisterCmd(&dgc.Command{
		Name:        "testcmd",
		Description: "List all server and its current state.",
		Usage:       "testcmd",
		IgnoreCase:  true,
		Handler:     ac.onlyAllowedRoles(ac.testCmdHandler),
	})
}

func NewAdminCmdProcessor(adminRoles []string) *adminCmdProcessor {
	return &adminCmdProcessor{
		adminRoles: adminRoles,
	}
}
