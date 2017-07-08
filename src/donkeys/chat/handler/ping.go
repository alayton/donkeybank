package handler

import (
	"time"

	"donkeys/chat/structs"
)

// Ping responds to PING commands with PONG
func Ping(channel *structs.ChannelSettings, msg structs.ServerMessage, conn chan string, throttle chan time.Time) {
	conn <- "PONG :" + msg.Text
}
