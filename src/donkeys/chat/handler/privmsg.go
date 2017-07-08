package handler

import (
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/boltdb/bolt"
	"github.com/spf13/viper"

	"donkeys/chat/structs"
)

var commands = map[string]func(channel *structs.ChannelSettings, msg structs.ServerMessage, conn chan string, throttle chan time.Time, rest string){
	"!commands":   help,
	"!mods":       mods,
	"!donkeys":    donkeys,
	"!gamble":     gamble,
	"!subonly":    subMode,
	"!emoteonly":  emoteMode,
	"!buycommand": buyCommand,

	"!mod":     mod,
	"!unmod":   unmod,
	"!give":    give,
	"!refresh": refresh,
}

// Privmsg parses PRIVMSG messages for commands
func Privmsg(channel *structs.ChannelSettings, msg structs.ServerMessage, conn chan string, throttle chan time.Time) {
	parts := strings.SplitN(msg.Text, " ", 2)
	command := strings.ToLower(parts[0])
	if cmd, ok := commands[command]; ok {
		rest := ""
		if len(parts) > 1 {
			rest = parts[1]
		}
		cmd(channel, msg, conn, throttle, rest)
	} else {
		command = strings.TrimPrefix(command, "!")
		for _, cmd := range channel.Commands {
			if cmd.Command == command && cmd.Expires.After(time.Now()) {
				boltdb := viper.Get("Bolt").(*bolt.DB)
				if err := boltdb.Update(func(tx *bolt.Tx) error {
					b := tx.Bucket([]byte(channel.Name))

					var count uint32
					countKey := fmt.Sprintf("command:%v", cmd.ID)
					buf := b.Get([]byte(countKey))
					if buf != nil {
						count = binary.LittleEndian.Uint32(buf)
					}
					count++

					message := strings.Replace(cmd.Message, "<count>", strconv.FormatUint(uint64(count), 10), -1)
					channel.Message(conn, throttle, message)

					buf = make([]byte, 4)
					binary.LittleEndian.PutUint32(buf, count)
					b.Put([]byte(countKey), buf)
					return nil
				}); err != nil {
					log.Error("Error with boltdb for custom command: ", err)
				}
			}
		}
	}
}
