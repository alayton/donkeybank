package commands

import (
	"fmt"
	"strings"
	"time"

	"donkeys/chat/structs"
)

func mods(channel *structs.ChannelSettings, msg structs.ServerMessage, conn chan string, throttle chan time.Time, rest string) {
	channel.Message(conn, throttle, fmt.Sprintf("Current %v moderators: %v", channel.CurrencyName, strings.Join(channel.Mods, ", ")))
}
