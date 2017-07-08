package commands

import (
	"encoding/binary"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/boltdb/bolt"
	"github.com/jmoiron/sqlx"
	"github.com/spf13/viper"

	"donkeys/chat/structs"
)

func buyCommand(channel *structs.ChannelSettings, msg structs.ServerMessage, conn chan string, throttle chan time.Time, rest string) {
	parts := strings.SplitN(rest, " ", 3)
	if len(parts) != 3 {
		return
	}

	command := parts[0]
	days, err := strconv.ParseUint(parts[1], 10, 64)
	if err != nil || days <= 0 {
		return
	}
	message := parts[2]

	if matches, err := regexp.MatchString("^[a-zA-Z0-9]{2,25}$", command); err != nil {
		log.Error("Error testing command name: ", err)
		return
	} else if !matches {
		return
	}
	price := days * channel.CommandCost
	message = strings.Trim(message, "/ ")
	if len(message) > 120 {
		return
	}

	mod := isMod(channel, msg)
	if mod {
		price = 0
	}

	db := viper.Get("Database").(*sqlx.DB)
	var count int
	if err := db.Get(&count, "SELECT COUNT(id) FROM commands WHERE `channel_id` = ? AND `trigger` = ? AND `expires` > ?", channel.ID, command, time.Now().Unix()); err != nil {
		log.Error("Error checking for duplicate command: ", err)
		return
	}

	if count > 0 {
		channel.Message(conn, throttle, fmt.Sprintf("That command is already in use (!%v)", command))
		return
	}

	boltdb := viper.Get("Bolt").(*bolt.DB)
	if err := boltdb.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(channel.Name))
		var donkeys uint64
		buf := b.Get([]byte(msg.Source))
		if buf != nil {
			donkeys = binary.LittleEndian.Uint64(buf)
		}

		if donkeys < price {
			channel.Message(conn, throttle, fmt.Sprintf("You don't have enough %v for that command, %v!", channel.CurrencyName, msg.Source))
			return nil
		}

		expires := time.Now().AddDate(0, 0, int(days))

		insert, err := db.Exec("INSERT INTO commands (`channel_id`, `trigger`, `message`, `expires`, `created_by`) VALUES (?, ?, ?, ?, ?)", channel.ID, command, message, expires.Unix(), msg.Source)
		if err != nil {
			return err
		}
		commandID, err := insert.LastInsertId()
		if err != nil {
			return err
		}

		donkeys -= price
		buf = make([]byte, 8)
		binary.LittleEndian.PutUint64(buf, donkeys)
		b.Put([]byte(msg.Source), buf)

		var count uint32
		buf = make([]byte, 4)
		binary.LittleEndian.PutUint32(buf, count)
		b.Put([]byte(fmt.Sprintf("command:%v", commandID)), buf)

		channel.Message(conn, throttle, fmt.Sprintf("New command added for %v days (costing %v %v): !%v", days, price, channel.CurrencyName, command))
		channel.NeedsUpdate <- true

		return nil
	}); err != nil {
		log.Error("Error with boltdb for !buycommand: ", err)
	}
}
