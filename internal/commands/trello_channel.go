package commands

import (
	"dgtrello/internal/core"
	"fmt"
	"time"

	"github.com/adlio/trello"
	"github.com/bwmarrin/discordgo"
	log "github.com/inconshreveable/log15"
)

var (
	eventEmbedColors = map[string]int{
		core.EventCreateCard:       0x27ae60, // green
		core.EventCopyCard:         0x27ae60, // cyan
		core.EventCommentCard:      0x7f8c8d, // gray
		core.EventDeleteCard:       0xe74c3c, // red
		core.EventUpdateCard:       0x2980b9, // carrot
		core.EventAddMemberToBoard: 0xf39c12, // orange
		core.EventAddMemberToCard:  0xf39c12, // orange
	}
)

type TrelloChannelConfig struct {
	ChannelId     string   `json:"channelId"`
	BoardId       string   `json:"boardId"`
	EnabledEvents []string `json:"enabledEvents"`
	LastActionId  string   `json:"lastActionId"`
}

type TrelloChannel struct {
	channelId string
	members   map[string]string
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

func (ch *TrelloChannel) renderCardEmbed(card *trello.Card, showCheckList bool) *discordgo.MessageEmbed {
	// Initialize embed message with name and description field
	fields := []*discordgo.MessageEmbedField{
		{
			Name:   fmt.Sprintf("🪧 %s", card.Name),
			Value:  card.Desc,
			Inline: false,
		},
	}
	// Add check lists field
	if showCheckList && len(card.Checklists) > 0 {
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
			if userId, exist := ch.members[member.Username]; exist {
				membersText += fmt.Sprintf("<@%s>", userId)
			} else {
				membersText += fmt.Sprintf("@%s ", member.Username)
			}
		}
	}
	fields = append(fields, &discordgo.MessageEmbedField{
		Name:   "👥 Assignees",
		Value:  membersText,
		Inline: false,
	})
	// Add due date field
	if card.Due != nil {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "🕒 Due date",
			Value:  card.Due.Local().Format(time.RFC1123),
			Inline: false,
		})
	}
	return &discordgo.MessageEmbed{
		URL:       card.ShortURL,
		Type:      "rich",
		Title:     card.Name,
		Timestamp: time.Now().Format(time.RFC3339),
		Fields:    fields,
	}
}

func (ch *TrelloChannel) handleEventUpdateCard(ctx *core.TrelloEventCtx, action *trello.Action) error {
	card, err := ch.fetchCard(ctx.Client, action.Data.Card.ID)
	if err != nil {
		return err
	}
	msg := ch.renderCardEmbed(card, true)
	msg.Color = eventEmbedColors[action.Type]
	msg.Title = fmt.Sprintf("%s update a card", action.MemberCreator.FullName)
	if card.Closed {
		msg.Title = fmt.Sprintf("%s archived a card", action.MemberCreator.FullName)
		msg.Color = eventEmbedColors[core.EventDeleteCard]
	} else if action.Data.ListBefore != nil && action.Data.ListAfter != nil {
		msg.Title = fmt.Sprintf(" %s moved a card to %s", action.MemberCreator.FullName, action.Data.ListAfter.Name)
	} else if action.Data.Old.Pos != 0 {
		// ignore card position update
		return nil
	}
	// Add board name
	msg.Title = fmt.Sprintf("%s - %s", msg.Title, action.Data.Board.Name)
	_, err = ch.session.ChannelMessageSendEmbed(ch.channelId, msg)
	return err
}

func (ch *TrelloChannel) handleEventCreateCard(ctx *core.TrelloEventCtx, action *trello.Action) error {
	card, err := ch.fetchCard(ctx.Client, action.Data.Card.ID)
	if err != nil {
		return err
	}
	msg := ch.renderCardEmbed(card, true)
	msg.Color = eventEmbedColors[action.Type]
	msg.Title = fmt.Sprintf("%s created a new card", action.MemberCreator.FullName)
	_, err = ch.session.ChannelMessageSendEmbed(ch.channelId, msg)
	return err
}

func (ch *TrelloChannel) handleEventDeleteCard(ctx *core.TrelloEventCtx, action *trello.Action) error {
	card, err := ch.fetchCard(ctx.Client, action.Data.Card.ID)
	if err != nil {
		return err
	}
	msg := ch.renderCardEmbed(card, true)
	msg.Color = eventEmbedColors[action.Type]
	msg.Title = fmt.Sprintf("%s deleted a card", action.MemberCreator.FullName)
	_, err = ch.session.ChannelMessageSendEmbed(ch.channelId, msg)
	return err
}

func (ch *TrelloChannel) handleEventCommentCard(ctx *core.TrelloEventCtx, action *trello.Action) error {
	card, err := ch.fetchCard(ctx.Client, action.Data.Card.ID)
	if err != nil {
		return err
	}
	msg := ch.renderCardEmbed(card, false)
	msg.Color = eventEmbedColors[action.Type]
	msg.Title = fmt.Sprintf("%s commented on a card", action.MemberCreator.FullName)
	msg.Fields = append(msg.Fields, &discordgo.MessageEmbedField{
		Name:   fmt.Sprintf("💬 %s commented", action.MemberCreator.FullName),
		Value:  truncateText(action.Data.Text, 1024), // maxLen 1024
		Inline: false,
	})
	_, err = ch.session.ChannelMessageSendEmbed(ch.channelId, msg)
	return err
}

func (ch *TrelloChannel) OnTrelloEvent(ctx *core.TrelloEventCtx, action *trello.Action) {
	var err error
	switch action.Type {
	case core.EventCreateCard:
		err = ch.handleEventCreateCard(ctx, action)
	case core.EventUpdateCard:
		err = ch.handleEventUpdateCard(ctx, action)
	case core.EventCopyCard:
		err = ch.handleEventCreateCard(ctx, action)
	case core.EventDeleteCard:
		err = ch.handleEventDeleteCard(ctx, action)
	case core.EventCommentCard:
		err = ch.handleEventCommentCard(ctx, action)
	}
	if err != nil {
		ch.session.ChannelMessageSend(ch.channelId, "❌ Internal error occurred, check log for more detail.")
		log.Error("Could not process board event", "actionId", action.ID, "error", err)
	}
}
