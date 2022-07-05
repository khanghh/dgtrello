package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
)

const (
	configFileName = "config.json"
)

type AppConfig struct {
	CmdPrefix     string   `json:"cmdPrefix"`
	DiscordToken  string   `json:"discordToken"`
	AdminRoles    []string `json:"adminRoles"`
	ServerID      string   `json:"serverID"`
	ChannelID     string   `json:"channelID"`
	BoardIDs      []string `json:"boardIDs"`
	PollInterval  int      `json:"pollInterval"`
	EnabledEvents []string `json:"enabledEvents"`
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
