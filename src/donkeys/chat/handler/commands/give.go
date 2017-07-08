package commands

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

func give(channel *structs.ChannelSettings, msg structs.ServerMessage, conn chan string, throttle chan time.Time, rest string) {
	if !isMod(channel, msg) {
		return
	}

	boltdb := viper.Get("Bolt").(*bolt.DB)

	parts := strings.SplitN(rest, " ", 3)
	target := strings.ToLower(parts[0])
	amount, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || amount == 0 {
		return
	}

	if err := boltdb.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(channel.Name))
		var donkeys uint64
		buf := b.Get([]byte(target))
		if buf != nil {
			donkeys = binary.LittleEndian.Uint64(buf)
		} else {
			return nil
		}
		sdonkeys := int64(donkeys)

		sdonkeys += amount
		if sdonkeys < 0 {
			sdonkeys = 0
		}
		buf = make([]byte, 8)
		binary.LittleEndian.PutUint64(buf, uint64(sdonkeys))
		b.Put([]byte(target), buf)

		channel.Message(conn, throttle, fmt.Sprintf("%v now has %v donkeys", target, sdonkeys))

		return nil
	}); err != nil {
		log.Error("Error with boltdb for !give: ", err)
	}
}
