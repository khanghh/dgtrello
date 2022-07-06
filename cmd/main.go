package main

import (
	"dgtrello/internal/command"
	"dgtrello/internal/discordbot"
	"dgtrello/internal/logger"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/jessevdk/go-flags"
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

	conf := MustLoadConfig()
	trelloProc, _ := command.NewTrelloCommandProcessor(conf.AdminRoles)
	bot, err := discordbot.NewDiscordBot(conf.DiscordToken, conf.ServerID, conf.ChannelID)
	if err != nil {
		log.Fatalln("Could not initialize discord bot.")
	}
	bot.SetCmdPrefix(conf.CmdPrefix)
	bot.AddProcessor(trelloProc)
	bot.Run()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	defer func() {
		<-sigCh
		os.Exit(1)
	}()
}
