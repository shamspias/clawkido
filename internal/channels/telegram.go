package channels

import (
	"clawkido/internal/config"
	"clawkido/internal/types"
	"context"
	"fmt"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// ReplyBufSize is the buffer size for reply channels.
// Must be large enough to hold responses from an entire handoff chain.
const ReplyBufSize = 16

// ReplyIdleTimeout is how long we wait for additional responses after
// the last received message before we stop listening. This allows
// handoff chains (manager → coder → reviewer) to complete.
const ReplyIdleTimeout = 90 * time.Second

// ReplyMaxWait is the absolute maximum time we wait for any response.
const ReplyMaxWait = 3 * time.Minute

type Telegram struct {
	token        string
	allowedUsers map[int64]bool // empty map = allow everyone
	inbox        chan<- types.Message
	logChan      chan<- types.LogEntry
}

func NewTelegram(cfg *config.Config, inbox chan<- types.Message, logs chan<- types.LogEntry) *Telegram {
	allowed := make(map[int64]bool, len(cfg.Telegram.AllowedUsers))
	for _, uid := range cfg.Telegram.AllowedUsers {
		allowed[uid] = true
	}
	return &Telegram{
		token:        cfg.Telegram.Token,
		allowedUsers: allowed,
		inbox:        inbox,
		logChan:      logs,
	}
}

func (t *Telegram) Name() string { return "Telegram" }

func (t *Telegram) Start(ctx context.Context) error {
	bot, err := tgbotapi.NewBotAPI(t.token)
	if err != nil {
		return fmt.Errorf("telegram connect: %w", err)
	}
	t.log("INFO", fmt.Sprintf("Connected as @%s", bot.Self.UserName))

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	for {
		select {
		case <-ctx.Done():
			bot.StopReceivingUpdates()
			t.log("INFO", "Disconnected (graceful)")
			return nil

		case update, ok := <-updates:
			if !ok {
				return nil
			}
			if update.Message == nil {
				continue
			}

			// Access control: if allowed_users is non-empty, enforce it.
			// Empty list = allow everyone.
			if len(t.allowedUsers) > 0 && !t.allowedUsers[update.Message.From.ID] {
				t.log("WARN", fmt.Sprintf("Blocked user %d (%s) — not in allowed_users",
					update.Message.From.ID, update.Message.From.UserName))
				continue
			}

			// Buffered channel so multiple agents in a handoff chain can write.
			replyChan := make(chan string, ReplyBufSize)

			t.inbox <- types.Message{
				Platform:  "Telegram",
				ChannelID: fmt.Sprintf("%d", update.Message.Chat.ID),
				Sender:    update.Message.From.UserName,
				Content:   update.Message.Text,
				ReplyChan: replyChan,
				Timestamp: time.Now(),
			}

			// Drain ALL responses from the handoff chain.
			go t.drainReplies(bot, update.Message.Chat.ID, replyChan)
		}
	}
}

// drainReplies reads from replyChan and sends each response as a separate
// Telegram message. It uses a rolling idle timeout: after each response,
// the timer resets. This lets handoff chains (agent A → agent B → ...)
// complete naturally without dropping responses.
func (t *Telegram) drainReplies(bot *tgbotapi.BotAPI, chatID int64, replyChan <-chan string) {
	deadline := time.NewTimer(ReplyMaxWait)
	defer deadline.Stop()

	idle := time.NewTimer(ReplyIdleTimeout)
	defer idle.Stop()

	count := 0
	for {
		select {
		case response, ok := <-replyChan:
			if !ok {
				// Channel closed, we're done.
				return
			}
			count++

			msg := tgbotapi.NewMessage(chatID, response)
			msg.ParseMode = "Markdown"
			if _, err := bot.Send(msg); err != nil {
				// Retry without markdown if parsing fails.
				msg.ParseMode = ""
				if _, err2 := bot.Send(msg); err2 != nil {
					t.log("ERROR", fmt.Sprintf("Send failed: %v", err2))
				}
			}

			// Reset idle timer — more responses may follow from handoffs.
			if !idle.Stop() {
				select {
				case <-idle.C:
				default:
				}
			}
			idle.Reset(ReplyIdleTimeout)

		case <-idle.C:
			// No new responses for ReplyIdleTimeout — handoff chain complete.
			if count == 0 {
				t.log("WARN", "No response received within idle timeout")
			}
			return

		case <-deadline.C:
			// Absolute timeout — safety net.
			if count == 0 {
				t.log("WARN", "Absolute timeout — no response received")
			}
			return
		}
	}
}

func (t *Telegram) log(level, msg string) {
	select {
	case t.logChan <- types.LogEntry{Level: level, Source: "Telegram", Message: msg, Time: time.Now()}:
	default:
	}
}
