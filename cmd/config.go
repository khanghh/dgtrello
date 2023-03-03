package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
)

const (
	defaultConfigFile = "config.json"
)

type ListenerConfig struct {
	ChannelId      string   `json:"channelId"`
	BoardId        string   `json:"boardId"`
	EnabledEvents  []string `json:"enabledEvents"`
	LastActivityId string   `json:"lastActivityId"`
}

type AppConfig struct {
	CmdPrefix    string           `json:"cmdPrefix"`
	DiscordToken string           `json:"discordToken"`
	AdminRoles   []string         `json:"adminRoles"`
	TrelloApiKey string           `json:"trelloApiKey"`
	TrelloToken  string           `json:"trelloToken"`
	PollInterval int              `json:"pollInterval"`
	Listeners    []ListenerConfig `json:"listeners"`
}

func loadConfig(cfgFile string) (*AppConfig, error) {
	buf, err := ioutil.ReadFile(cfgFile)
	if err != nil {
		return nil, err
	}
	conf := &AppConfig{}
	if err = json.Unmarshal(buf, conf); err != nil {
		return nil, err
	}
	return conf, nil
}

func MustLoadConfig(cfgFile string) *AppConfig {
	conf, err := loadConfig(cfgFile)
	if err != nil {
		log.Fatalln("Could not parse json config file.")
	}
	return conf
}
