package commands

import (
	"time"

	"donkeys/chat/structs"
)

// Refresh forces the bot to update settings from the database
func Refresh(channel *structs.ChannelSettings, msg structs.ServerMessage, conn chan string, throttle chan time.Time, rest string) {
	if !isMod(channel, msg) {
		return
	}
	channel.NeedsUpdate <- true
}
