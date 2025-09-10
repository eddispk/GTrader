package bot

import (
	"strconv"
	"strings"

	"bot/data"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Sends a message to a channel/group by @username OR numeric chat id.
func SendToChannel(order *data.Bot, idOrUser string, text string) {
	if order == nil || order.Botapi == nil || idOrUser == "" || text == "" {
		return
	}

	// numeric chat id?
	if !strings.HasPrefix(idOrUser, "@") {
		if id, err := strconv.ParseInt(strings.TrimSpace(idOrUser), 10, 64); err == nil {
			msg := tgbotapi.NewMessage(id, text)
			msg.ParseMode = "HTML"
			_, _ = order.Botapi.Send(msg)
			return
		}
	}

	// else use @channel username
	ch := idOrUser
	if !strings.HasPrefix(ch, "@") {
		ch = "@" + ch
	}
	msg := tgbotapi.NewMessageToChannel(ch, text)
	msg.ParseMode = "HTML"
	_, _ = order.Botapi.Send(msg)
}
