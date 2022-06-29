package main

import (
	"dgtrello/internal/discordbot"
	"dgtrello/internal/logger"
	"log"
	"os"

	"github.com/jessevdk/go-flags"
	"github.com/lus/dgc"
)

type CommandOptions struct {
	Version VersionCmd `command:"version" description:"Prints version information"`
}

func init() {
	if os.Getenv("DEBUG") != "" {
		logger.SetLogLevel(logger.LevelDebug)
	}
}

func main() {
	opts := &CommandOptions{}
	parser := flags.NewParser(opts, flags.Default)
	parser.SubcommandsOptional = true
	_, err := parser.Parse()
	if err != nil {
		switch flagsErr := err.(type) {
		case *flags.Error:
			if flagsErr.Type == flags.ErrHelp {
				os.Exit(0)
			}
		default:
			log.Fatalln(err)
		}
	}
	bot := discordbot.DiscordBot{}
	bot.RegisterCmd(&dgc.Command{}, nil)
	bot.Run()
}
