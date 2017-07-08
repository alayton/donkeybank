package commands

import (
	"time"

	"donkeys/chat/structs"
)

func refresh(channel *structs.ChannelSettings, msg structs.ServerMessage, conn chan string, throttle chan time.Time, rest string) {
	if !isMod(channel, msg) {
		return
	}
	channel.NeedsUpdate <- true
}
