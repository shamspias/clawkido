package channels

import (
	"clawkido/internal/types"
	"time"

	"github.com/bwmarrin/discordgo"
)

type Discord struct {
	token   string
	inbox   chan<- types.Message
	logChan chan<- types.LogEntry
	session *discordgo.Session
}

func NewDiscord(token string, inbox chan<- types.Message, logs chan<- types.LogEntry) *Discord {
	return &Discord{token: token, inbox: inbox, logChan: logs}
}

func (d *Discord) Start() {
	var err error
	d.session, err = discordgo.New("Bot " + d.token)
	if err != nil {
		d.log("ERROR", err.Error())
		return
	}

	d.session.AddHandler(d.messageCreate)

	if err := d.session.Open(); err != nil {
		d.log("ERROR", "Connection failed: "+err.Error())
		return
	}
	d.log("INFO", "Connected to Gateway")
}

func (d *Discord) messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore self
	if m.Author.ID == s.State.User.ID {
		return
	}

	replyChan := make(chan string)

	d.inbox <- types.Message{
		Platform:  "Discord",
		Sender:    m.Author.Username,
		Content:   m.Content,
		ReplyChan: replyChan,
	}

	go func(channelID string) {
		response := <-replyChan
		s.ChannelMessageSend(channelID, response)
	}(m.ChannelID)
}

func (d *Discord) log(level, msg string) {
	d.logChan <- types.LogEntry{Level: level, Source: "Discord", Message: msg, Time: time.Now()}
}
