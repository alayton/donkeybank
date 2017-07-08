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

// EmoteOnly puts the channel in emote only mode for two minutes, if modded
func EmoteOnly(channel *structs.ChannelSettings, msg structs.ServerMessage, conn chan string, throttle chan time.Time, rest string) {
	boltdb := viper.Get("Bolt").(*bolt.DB)

	if err := boltdb.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(channel.Name))
		var donkeys uint64
		buf := b.Get([]byte(msg.Source))
		if buf != nil {
			donkeys = binary.LittleEndian.Uint64(buf)
		}

		if donkeys < channel.EmoteModeCost {
			return nil
		}

		gocache := viper.Get("Cache").(*cache.Cache)
		cooldownKey := fmt.Sprintf("emoteonly:%v", channel.Name)
		if readyTime, ok := gocache.Get(cooldownKey); ok {
			ready := readyTime.(int64)
			if ready > time.Now().Unix() {
				return nil
			}
		}
		gocache.Set(cooldownKey, time.Now().Add(time.Minute*30).Unix(), cache.DefaultExpiration)

		donkeys -= channel.EmoteModeCost
		buf = make([]byte, 8)
		binary.LittleEndian.PutUint64(buf, donkeys)
		b.Put([]byte(msg.Source), buf)

		channel.Message(conn, throttle, "/emoteonly")
		channel.Message(conn, throttle, fmt.Sprintf("%v put chate in emote only mode for 2 minutes!", msg.Source))
		time.AfterFunc(time.Minute*2, func() {
			channel.Message(conn, throttle, "/emoteonlyoff")
		})

		return nil
	}); err != nil {
		log.Error("Error with boltdb for !emoteonly: ", err)
	}
}
