package commands

import (
	"encoding/binary"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/boltdb/bolt"
	"github.com/patrickmn/go-cache"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/alayton/donkeybank/chat/structs"
)

// Gamble allows users to increase their currency through gambling (!gamble <amount_to_bet>)
func Gamble(channel *structs.ChannelSettings, msg structs.ServerMessage, conn chan string, throttle chan time.Time, rest string) {
	gocache := viper.Get("Cache").(*cache.Cache)
	cooldownKey := fmt.Sprintf("gamble:%v:%v", channel.Name, msg.Source)
	if readyTime, ok := gocache.Get(cooldownKey); ok {
		ready := readyTime.(int64)
		if ready > time.Now().Unix() {
			channel.Message(conn, throttle, fmt.Sprintf("%v has to wait %v before gambling again", msg.Source, time.Unix(ready, 0).Round(time.Second).Sub(time.Now().Round(time.Second))))
			return
		}
	}
	gocache.Set(cooldownKey, time.Now().Add(time.Minute*5).Unix(), cache.DefaultExpiration)

	boltdb := viper.Get("Bolt").(*bolt.DB)

	parts := strings.SplitN(rest, " ", 2)
	bet, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || bet <= 0 {
		return
	}

	roll := rand.Int31n(100) + 1
	var factor int64
	if roll <= 1 {
		factor = -2
	} else if roll <= 60 {
		factor = -1
	} else if roll <= 98 {
		factor = 1
	} else {
		factor = 2
	}
	var sdonkeys int64

	if err := boltdb.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(channel.Name))
		var donkeys uint64
		buf := b.Get([]byte(msg.Source))
		if buf != nil {
			donkeys = binary.LittleEndian.Uint64(buf)
		}

		sdonkeys = int64(donkeys)

		if bet > sdonkeys {
			bet = sdonkeys
		}
		sdonkeys += bet * factor
		if sdonkeys < 0 {
			sdonkeys = 0
		}

		buf = make([]byte, 8)
		binary.LittleEndian.PutUint64(buf, uint64(sdonkeys))
		b.Put([]byte(msg.Source), buf)

		return nil
	}); err != nil {
		log.Error("Error with boltdb for !gamble: ", err)
	}

	result := "won"
	change := bet * factor
	if factor < 0 {
		result = "lost"
		change *= -1
	}
	channel.Message(conn, throttle, fmt.Sprintf("Rolled %v. %v %v %v %v and now has %v %v", roll, msg.Source, result, change, channel.CurrencyName, sdonkeys, channel.CurrencyName))
}
