package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
)

const (
	configFileName = "config.json"
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

func loadConfig() (*AppConfig, error) {
	buf, err := ioutil.ReadFile(configFileName)
	if err != nil {
		return nil, err
	}
	conf := &AppConfig{}
	if err = json.Unmarshal(buf, conf); err != nil {
		return nil, err
	}
	return conf, nil
}

func MustLoadConfig() *AppConfig {
	conf, err := loadConfig()
	if err != nil {
		log.Fatalln("Could not parse json config file.")
	}
	return conf
}
