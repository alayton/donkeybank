package commands

import (
	"encoding/binary"
	"fmt"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/boltdb/bolt"
	"github.com/patrickmn/go-cache"
	"github.com/spf13/viper"

	"donkeys/chat/structs"
)

// SubOnly puts the channel in sub only mode for two minutes, if modded
func SubOnly(channel *structs.ChannelSettings, msg structs.ServerMessage, conn chan string, throttle chan time.Time, rest string) {
	boltdb := viper.Get("Bolt").(*bolt.DB)

	if err := boltdb.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(channel.Name))
		var donkeys uint64
		buf := b.Get([]byte(msg.Source))
		if buf != nil {
			donkeys = binary.LittleEndian.Uint64(buf)
		}

		if donkeys < channel.SubModeCost {
			return nil
		}

		gocache := viper.Get("Cache").(*cache.Cache)
		cooldownKey := fmt.Sprintf("subonly:%v", channel.Name)
		if readyTime, ok := gocache.Get(cooldownKey); ok {
			ready := readyTime.(int64)
			if ready > time.Now().Unix() {
				return nil
			}
		}
		gocache.Set(cooldownKey, time.Now().Add(time.Minute*30).Unix(), cache.DefaultExpiration)

		donkeys -= channel.SubModeCost
		buf = make([]byte, 8)
		binary.LittleEndian.PutUint64(buf, donkeys)
		b.Put([]byte(msg.Source), buf)

		channel.Message(conn, throttle, "/subscribers")
		channel.Message(conn, throttle, fmt.Sprintf("%v put chat in sub only mode for 2 minutes!", msg.Source))
		time.AfterFunc(time.Minute*2, func() {
			channel.Message(conn, throttle, "/subscribersoff")
		})

		return nil
	}); err != nil {
		log.Error("Error with boltdb for !subonly: ", err)
	}
}
