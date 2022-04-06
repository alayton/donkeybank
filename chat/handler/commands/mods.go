package commands

import (
	"fmt"
	"strings"
	"time"

	"github.com/alayton/donkeybank/chat/structs"
)

// Mods lists the bot's current mods
func Mods(channel *structs.ChannelSettings, msg structs.ServerMessage, conn chan string, throttle chan time.Time, rest string) {
	channel.Message(conn, throttle, fmt.Sprintf("Current %v moderators: %v", channel.CurrencyName, strings.Join(channel.Mods, ", ")))
}
