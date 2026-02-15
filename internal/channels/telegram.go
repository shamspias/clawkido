package channels

import (
	"clawkido/internal/types"
	"fmt"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Telegram struct {
	token   string
	inbox   chan<- types.Message
	logChan chan<- types.LogEntry
	bot     *tgbotapi.BotAPI
}

func NewTelegram(token string, inbox chan<- types.Message, logs chan<- types.LogEntry) *Telegram {
	return &Telegram{token: token, inbox: inbox, logChan: logs}
}

func (t *Telegram) Start() {
	var err error
	t.bot, err = tgbotapi.NewBotAPI(t.token)
	if err != nil {
		t.log("ERROR", "Failed to connect: "+err.Error())
		return
	}
	t.log("INFO", fmt.Sprintf("Connected as %s", t.bot.Self.UserName))

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := t.bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		// Create a channel for the engine to reply back to this specific message
		replyChan := make(chan string)

		// Send to Engine
		t.inbox <- types.Message{
			Platform:  "Telegram",
			Sender:    update.Message.From.UserName,
			Content:   update.Message.Text,
			ReplyChan: replyChan,
		}

		// Listen for the Engine's response in a background goroutine
		go func(chatID int64) {
			response := <-replyChan
			msg := tgbotapi.NewMessage(chatID, response)
			msg.ParseMode = "Markdown"
			t.bot.Send(msg)
		}(update.Message.Chat.ID)
	}
}

func (t *Telegram) log(level, msg string) {
	t.logChan <- types.LogEntry{Level: level, Source: "Telegram", Message: msg, Time: time.Now()}
}
