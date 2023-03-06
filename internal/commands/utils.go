package commands

import "fmt"

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

func getCardUrl(idModel string) string {
	return fmt.Sprintf("%s/c/%s", trelloUrl, idModel)
}

func getBoardUrl(idModel string) string {
	return fmt.Sprintf("%s/b/%s", trelloUrl, idModel)
}
