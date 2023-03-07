package commands

import "encoding/json"

const (
	trelloUrl = "https://trello.com"
)

var (
	labelColors = map[string]int{
		"green":  0x7bc86c,
		"yellow": 0xf5dd29,
		"orange": 0xffaf3f,
		"red":    0xef7564,
		"purple": 0xCD8DE5,
		"blue":   0x5ba4cf,
		"sky":    0x29cce5,
		"lime":   0x6deca9,
		"pink":   0xff8ed4,
		"black":  0x344563,
	}
)

func unmarshalToMap(data []byte) (map[string]interface{}, error) {
	ret := make(map[string]interface{})
	if err := json.Unmarshal(data, &ret); err != nil {
		return nil, err
	}
	return ret, nil
}

func parseUserId(str string) (string, bool) {
	if len(str) > 3 && str[0:2] == "<@" && str[len(str)-1:] == ">" {
		return str[2 : len(str)-1], true
	}
	return "", false
}

func truncateText(str string, maxLen uint) string {
	if len(str) <= int(maxLen) {
		return str
	}
	return str[0:maxLen-3] + "..."
}
