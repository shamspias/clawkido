package channels

import (
	"clawkido/internal/types"
	"context"
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
)

type Discord struct {
	token   string
	inbox   chan<- types.Message
	logChan chan<- types.LogEntry
}

func NewDiscord(token string, inbox chan<- types.Message, logs chan<- types.LogEntry) *Discord {
	return &Discord{token: token, inbox: inbox, logChan: logs}
}

func (d *Discord) Name() string { return "Discord" }

func (d *Discord) Start(ctx context.Context) error {
	session, err := discordgo.New("Bot " + d.token)
	if err != nil {
		return fmt.Errorf("discord session: %w", err)
	}

	session.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		if m.Author.ID == s.State.User.ID {
			return
		}

		replyChan := make(chan string, ReplyBufSize)

		d.inbox <- types.Message{
			Platform:  "Discord",
			ChannelID: m.ChannelID,
			Sender:    m.Author.Username,
			Content:   m.Content,
			ReplyChan: replyChan,
			Timestamp: time.Now(),
		}

		// Drain all responses from the handoff chain.
		go d.drainReplies(s, m.ChannelID, replyChan)
	})

	if err := session.Open(); err != nil {
		return fmt.Errorf("discord connect: %w", err)
	}
	d.log("INFO", "Connected to Gateway")

	<-ctx.Done()
	d.log("INFO", "Disconnecting (graceful)")
	return session.Close()
}

// drainReplies reads from replyChan with a rolling idle timeout,
// sending each response as a separate Discord message.
func (d *Discord) drainReplies(s *discordgo.Session, channelID string, replyChan <-chan string) {
	deadline := time.NewTimer(ReplyMaxWait)
	defer deadline.Stop()

	idle := time.NewTimer(ReplyIdleTimeout)
	defer idle.Stop()

	for {
		select {
		case response, ok := <-replyChan:
			if !ok {
				return
			}
			if _, err := s.ChannelMessageSend(channelID, response); err != nil {
				d.log("ERROR", fmt.Sprintf("Send failed: %v", err))
			}

			// Reset idle timer.
			if !idle.Stop() {
				select {
				case <-idle.C:
				default:
				}
			}
			idle.Reset(ReplyIdleTimeout)

		case <-idle.C:
			return
		case <-deadline.C:
			return
		}
	}
}

func (d *Discord) log(level, msg string) {
	select {
	case d.logChan <- types.LogEntry{Level: level, Source: "Discord", Message: msg, Time: time.Now()}:
	default:
	}
}
