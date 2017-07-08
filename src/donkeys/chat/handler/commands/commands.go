package commands

import (
	"fmt"
	"time"

	"donkeys/chat/structs"
)

func commands(channel *structs.ChannelSettings, msg structs.ServerMessage, conn chan string, throttle chan time.Time, rest string) {
	channel.Message(conn, throttle, fmt.Sprintf("Available commands: !donkeys, !gamble <amount>, !buycommand <command> <days> <message> (%v %v per day)", channel.CommandCost, channel.CurrencyName))
}
