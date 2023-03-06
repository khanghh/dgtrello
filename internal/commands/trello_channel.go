package commands

import (
	"dgtrello/internal/core"
	"encoding/json"
	"fmt"
	"time"

	"github.com/adlio/trello"
	"github.com/bwmarrin/discordgo"
	log "github.com/inconshreveable/log15"
)

type TrelloChannelConfig struct {
	ChannelId     string   `json:"channelId"`
	BoardId       string   `json:"boardId"`
	EnabledEvents []string `json:"enabledEvents"`
	LastActionId  string   `json:"lastActionId"`
}

type TrelloChannel struct {
	channelId string
	session   *discordgo.Session
	listener  *core.TrelloEventListener
}

func (ch *TrelloChannel) BoardId() string {
	return ch.listener.IdModel
}

func (ch *TrelloChannel) ChannelId() string {
	return ch.channelId
}

func (ch *TrelloChannel) fetchCard(client *trello.Client, cardId string) (*trello.Card, error) {
	return client.GetCard(cardId, trello.Arguments{
		"members":         "true",
		"member_fields":   "username",
		"checklists":      "all",
		"checkItemStates": "false",
	})
}

func (ch *TrelloChannel) renderCardEmbed(card *trello.Card) *discordgo.MessageEmbed {
	embedColor := 0x0099ff
	if len(card.Labels) > 0 {
		label := card.Labels[0]
		embedColor = labelColors[label.Color]
	}
	// Initialize embed message with name and description field
	fields := []*discordgo.MessageEmbedField{
		{
			Name:   fmt.Sprintf("🪧 %s", card.Name),
			Value:  card.Desc,
			Inline: false,
		},
	}
	// Add check lists field
	if len(card.Checklists) > 0 {
		for _, checklist := range card.Checklists {
			itemsMsg := ""
			for _, item := range checklist.CheckItems {
				if item.State == "complete" {
					itemsMsg += fmt.Sprintf("✅ %s\n", item.Name)
				} else {
					itemsMsg += fmt.Sprintf("⭕️ %s\n", item.Name)
				}
			}
			fields = append(fields, &discordgo.MessageEmbedField{
				Name:   "📝 " + checklist.Name,
				Value:  itemsMsg,
				Inline: false,
			})
		}
	}
	// Add assignees filed
	membersText := "Not assigned yet"
	if len(card.Members) > 0 {
		membersText = ""
		for _, member := range card.Members {
			membersText += fmt.Sprintf("<@%s>", member.Username)
		}
	}
	fields = append(fields, &discordgo.MessageEmbedField{
		Name:   "👥 Assignees",
		Value:  membersText,
		Inline: false,
	})
	// Add due date field
	dueTimeText := "No due time"
	if card.Due != nil {
		dueTimeText = card.Due.String()
	}
	fields = append(fields, &discordgo.MessageEmbedField{
		Name:   "🕒 Due date",
		Value:  dueTimeText,
		Inline: false,
	})
	return &discordgo.MessageEmbed{
		URL:       card.ShortURL,
		Type:      "rich",
		Title:     card.Name,
		Timestamp: time.Now().Format(time.RFC3339),
		Fields:    fields,
		Color:     embedColor,
	}
}

func printJSON(val interface{}) {
	buf, _ := json.MarshalIndent(val, "", "  ")
	fmt.Println(string(buf))
}

func (ch *TrelloChannel) handleCardUpdate(ctx *core.TrelloEventCtx, action *trello.Action) error {
	card, err := ch.fetchCard(ctx.Client, action.Data.Card.ID)
	if err != nil {
		return err
	}
	msg := ch.renderCardEmbed(card)
	msg.Title = action.ID
	if action.Data.ListBefore != nil && action.Data.ListAfter != nil {
		msg.Title = fmt.Sprintf(" %s moved a card to %s", action.MemberCreator.FullName, action.Data.ListAfter.Name)
	}
	msg.Title = fmt.Sprintf("%s - %s", msg.Title, action.Data.Board.Name)
	_, err = ch.session.ChannelMessageSendEmbed(ch.channelId, msg)
	return err
}

func (ch *TrelloChannel) OnTrelloEvent(ctx *core.TrelloEventCtx, action *trello.Action) {
	var err error
	if action.Type == core.EventUpdateCard {
		err = ch.handleCardUpdate(ctx, action)
	}
	// printJSON(action)
	if err != nil {
		ch.session.ChannelMessageSend(ch.channelId, "❌ Internal error occured, check log for more detail.")
		log.Error("Could not process board event", "actionId", action.ID, "error", err)
	}
}
