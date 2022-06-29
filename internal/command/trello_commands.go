package discordbot

import (
	"github.com/lus/dgc"
)

type TrelloCmdProcessor struct {
}

func (ac *TrelloCmdProcessor) RegisterCommands(cmdRouter *dgc.Router) {
	cmdRouter.RegisterCmd(commands.PingCommand)
}

func newTrelloCommandProcessor() (*TrelloCmdProcessor, error) {
	return &TrelloCmdProcessor{}, nil
}
