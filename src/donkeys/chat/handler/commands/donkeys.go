package commands

import (
	"encoding/binary"
	"fmt"
	"strconv"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/boltdb/bolt"
	"github.com/spf13/viper"

	"donkeys/chat/structs"
)

func donkeys(channel *structs.ChannelSettings, msg structs.ServerMessage, conn chan string, throttle chan time.Time, rest string) {
	boltdb := viper.Get("Bolt").(*bolt.DB)

	if err := boltdb.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(channel.Name))
		var donkeys uint64
		buf := b.Get([]byte(msg.Source))
		if buf != nil {
			donkeys = binary.LittleEndian.Uint64(buf)
		}

		channel.Message(conn, throttle, fmt.Sprintf("%v has %v donkeys", msg.Source, strconv.FormatUint(donkeys, 10)))
		return nil
	}); err != nil {
		log.Error("Error reading from boltdb for !donkeys: ", err)
	}
}
