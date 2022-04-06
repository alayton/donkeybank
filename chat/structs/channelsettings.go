package structs

import (
	"fmt"
	"time"
)

// Message sends a message to this channel
func (c *ChannelSettings) Message(conn chan string, throttle chan time.Time, message string) {
	<-throttle
	conn <- fmt.Sprintf("PRIVMSG #%v :%v", c.Name, message)
}

// ChannelSettings holds the settings for the channel..
type ChannelSettings struct {
	ID            int64            `db:"id"`
	Name          string           `db:"name"`
	Enabled       bool             `db:"enabled"`
	GainRate      uint64           `db:"gain_rate"`
	OnlineText    string           `db:"online_text"`
	OfflineText   string           `db:"offline_text"`
	CurrencyName  string           `db:"currency_name"`
	SubModeCost   uint64           `db:"sub_mode_cost"`
	EmoteModeCost uint64           `db:"emote_mode_cost"`
	CommandCost   uint64           `db:"command_cost"`
	ModList       string           `db:"mod_list"`
	Mods          []string         `db:"-"`
	Commands      []ChannelCommand `db:"-"`
	NeedsUpdate   chan bool        `db:"-"`
}

// ChannelCommand holds purchased commands for channels
type ChannelCommand struct {
	ID         int64     `db:"id"`
	ChannelID  int64     `db:"channel_id"`
	Command    string    `db:"trigger"`
	Message    string    `db:"message"`
	ExpireTime int64     `db:"expires"`
	CreatedBy  string    `db:"created_by"`
	Expires    time.Time `db:"-"`
}
