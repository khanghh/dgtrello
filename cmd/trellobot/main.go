package main

import (
	"context"
	"dgtrello/internal/commands"
	"dgtrello/internal/core"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/adlio/trello"
	log "github.com/inconshreveable/log15"
	"github.com/urfave/cli/v2"
)

var (
	app       *cli.App
	gitCommit string
	gitDate   string
	gitTag    string
)

func init() {
	app = cli.NewApp()
	app.Name = "Trello bot"
	app.EnableBashCompletion = true
	app.Usage = "Discord bot notify trello board events"
	app.Flags = []cli.Flag{
		configFileFlag,
		verbosityFlag,
	}
	app.Action = run
}

func initLogger(verbosity int) {
	log.Root().SetHandler(log.MultiHandler(
		log.StreamHandler(os.Stderr, log.TerminalFormat()),
		log.Must.FileHandler("bot.log", log.LogfmtFormat())),
	)
}

func printGreeting() {
	msgArr := []string{app.Name}
	if len(gitTag) > 0 {
		msgArr = append(msgArr, gitTag)
	} else if len(gitCommit) > 0 {
		msgArr = append(msgArr, gitCommit)
	}
	log.Info(strings.Join(msgArr, " - "))
}

func runBot(bot *core.DiscordBot) {
	printGreeting()
	ctx, cancel := context.WithCancel(context.Background())
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	go func() {
		<-sigCh
		cancel()
	}()
	bot.Run(ctx)
}

func run(ctx *cli.Context) error {
	initLogger(ctx.Int(verbosityFlag.Name))
	configFile := ctx.String(configFileFlag.Name)
	conf := MustLoadConfig(configFile)
	trelloClient := trello.NewClient(conf.TrelloApiKey, conf.TrelloToken)
	pollInterval := time.Duration(conf.PollInterval) * time.Millisecond
	trelloEventHub := core.NewTrelloEventHub(trelloClient, pollInterval)
	trelloProc, err := commands.NewTrelloCommandProcessor(configFile, trelloEventHub)
	if err != nil {
		log.Crit("Cout not initialize trello command.")
	}
	trelloProc.SetAllowedRoles(conf.AdminRoles)
	bot, err := core.NewDiscordBot(conf.DiscordToken, conf.CmdPrefix)
	if err != nil {
		log.Crit("Could not initialize discord bot.")
	}
	bot.AddCommandProcessor(trelloProc)
	runBot(bot)
	return nil
}

func main() {
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
