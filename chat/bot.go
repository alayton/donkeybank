package chat

import (
	"bufio"
	"net"
	"net/textproto"
	"regexp"
	"strings"
	"time"

	"github.com/boltdb/bolt"
	"github.com/jmoiron/sqlx"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/alayton/donkeybank/chat/handler"
	"github.com/alayton/donkeybank/chat/structs"
)

/*
	match[1] = source
	match[2] = command
	match[3] = target
	match[4] = subcommand
	match[5] = text
*/
//                                       source                 command       target         subcommand     text
var respRegex = regexp.MustCompile("(?::([^! ]+)(?:![^ ]+)?)? *([A-Z0-9]+)? *([^: ]+)? *=? *([^: ]+)? *(?::(.+))?")

var commandHandlers = map[string]func(*structs.ChannelSettings, structs.ServerMessage, chan string, chan time.Time){
	"PING":    handler.Ping,
	"PRIVMSG": handler.Privmsg,
}

// Run connects to the twitch IRC server
func Run(db *sqlx.DB) {
	rows, err := db.Queryx("SELECT * FROM channels WHERE enabled = 1")
	if err != nil {
		log.Error("Failed to select channels: ", err)
		return
	}
	defer rows.Close()

	now := time.Now().Unix()
	for rows.Next() {
		channel := &structs.ChannelSettings{}
		if err := rows.StructScan(channel); err != nil {
			log.Error("Channel scan error: ", err)
			continue
		} else if !channel.Enabled {
			continue
		}
		channel.Mods = strings.Split(channel.ModList, ",")

		if err := db.Select(&channel.Commands, "SELECT * FROM commands WHERE channel_id = ? AND expires > ?", channel.ID, now); err != nil {
			log.Error("Error querying for channel commands: ", err)
		}

		for key, cmd := range channel.Commands {
			channel.Commands[key].Expires = time.Unix(cmd.ExpireTime, 0)
		}

		go connect(db, channel)
		time.Sleep(1 * time.Second)
	}
}

func connect(db *sqlx.DB, channel *structs.ChannelSettings) {
	conn, err := net.Dial("tcp", "irc.chat.twitch.tv:6667")
	if err != nil {
		log.Error("Failed to connect to IRC server: ", err)
		go reconnect(db, channel)
		return
	}
	defer conn.Close()

	boltdb := viper.Get("Bolt").(*bolt.DB)
	if err := boltdb.Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists([]byte(channel.Name)); err != nil {
			return err
		}
		return nil
	}); err != nil {
		log.Error("Failed to create bucket for ", channel.Name, ": ", err)
		return
	}

	conn.Write([]byte("PASS " + viper.GetString("TwitchChatPassword") + "\r\n"))
	conn.Write([]byte("NICK " + viper.GetString("TwitchChatUsername") + "\r\n"))
	//conn.Write([]byte("CAP REQ :twitch.tv/membership\r\n"))
	conn.Write([]byte("JOIN #" + channel.Name + "\r\n"))

	r := textproto.NewReader(bufio.NewReader(conn))

	throttleTicker := time.NewTicker((time.Second * 30) / 20)
	defer throttleTicker.Stop()
	throttle := make(chan time.Time)
	go func() {
		for t := range throttleTicker.C {
			select {
			case throttle <- t:
			default:
			}
		}
	}()

	writeChannel := make(chan string)
	go func() {
		for {
			line := <-writeChannel
			conn.Write([]byte(line + "\r\n"))
		}
	}()

	channel.NeedsUpdate = make(chan bool)
	defer func() {
		close(channel.NeedsUpdate)
	}()
	go func() {
		for {
			_, ok := <-channel.NeedsUpdate
			if !ok {
				return
			}

			db.Get(&channel, "SELECT * FROM channels WHERE id = ?", channel.ID)
			channel.Mods = strings.Split(channel.ModList, ",")

			if err := db.Select(&channel.Commands, "SELECT * FROM commands WHERE channel_id = ? AND expires > ?", channel.ID, time.Now().Unix()); err != nil {
				log.Error("Error querying for channel commands: ", err)
			}

			for key, cmd := range channel.Commands {
				channel.Commands[key].Expires = time.Unix(cmd.ExpireTime, 0)
			}
		}
	}()

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	go func() {
		channelTick(db, boltdb, channel, writeChannel, throttle)
		for range ticker.C {
			channelTick(db, boltdb, channel, writeChannel, throttle)
		}
	}()

	log.Info("Connected to " + channel.Name)

	for {
		msg, err := r.ReadLine()
		if err != nil {
			log.Error("ReadLine error: ", err)
			go reconnect(db, channel)
			return
		}

		matches := respRegex.FindStringSubmatch(msg)
		if len(matches) == 0 {
			continue
		}

		if f, ok := commandHandlers[matches[2]]; ok {
			msg := structs.ServerMessage{
				Source:     matches[1],
				Command:    matches[2],
				Target:     matches[3],
				Subcommand: matches[4],
				Text:       matches[5],
			}
			f(channel, msg, writeChannel, throttle)
		}
	}
}

func reconnect(db *sqlx.DB, channel *structs.ChannelSettings) {
	time.Sleep(time.Second * 10)
	connect(db, channel)
}
