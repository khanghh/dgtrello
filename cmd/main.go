package main

import (
	"context"
	"dgtrello/internal/commands"
	"dgtrello/internal/core"
	"dgtrello/internal/discordbot"
	"dgtrello/internal/logger"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/adlio/trello"
	"github.com/jessevdk/go-flags"
)

type CommandOptions struct {
	Config  string     `short:"c" long:"config" description:"JSON config file"`
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

	ctx, cancel := context.WithCancel(context.Background())
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	go func() {
		<-sigCh
		cancel()
	}()

	if len(opts.Config) == 0 {
		opts.Config = defaultConfigFile
	}
	conf := MustLoadConfig(opts.Config)
	trelloClient := trello.NewClient(conf.TrelloApiKey, conf.TrelloToken)
	pollInterval := time.Duration(conf.PollInterval) * time.Millisecond
	trelloEventHub := core.NewTrelloEventHub(trelloClient, pollInterval)
	trelloProc, err := commands.NewTrelloCommandProcessor(opts.Config, trelloEventHub)
	if err != nil {
		logger.Fatalln("Cout not initialize trello command.")
	}
	trelloProc.SetAllowedRoles(conf.AdminRoles)
	bot, err := discordbot.NewDiscordBot(conf.DiscordToken)
	if err != nil {
		logger.Fatalln("Could not initialize discord bot.")
	}
	bot.SetCmdPrefix(conf.CmdPrefix)
	bot.AddCommandProcessor(trelloProc)
	printGreeting()
	bot.Run(ctx)
}
