package chat

import (
	"encoding/binary"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/boltdb/bolt"
	"github.com/jmoiron/sqlx"
	"github.com/patrickmn/go-cache"
	"github.com/spf13/viper"

	"donkeys/chat/structs"
)

type streamStatus struct {
	Stream *struct {
		ID      int64 `json:"_id"`
		Channel struct {
			Name string `json:"name"`
		} `json:"channel"`
	} `json:"stream"`
}

var tickClient = &http.Client{Timeout: 15 * time.Second}

type chatterList struct {
	Count  int                 `json:"chatter_count"`
	Groups map[string][]string `json:"chatters"`
}

func channelTick(db *sqlx.DB, boltdb *bolt.DB, channel *structs.ChannelSettings, conn chan string, throttle chan time.Time) {
	r, err := tickClient.Get("https://api.twitch.tv/kraken/streams/" + strconv.FormatInt(channel.ID, 10) + "?stream_type=live&api_version=5&client_id=" + viper.GetString("TwitchClientID"))
	if err != nil {
		log.Error("Error fetching stream status for ", channel.Name, " (", channel.ID, "): ", err)
		return
	}
	defer r.Body.Close()

	var strm streamStatus
	err = json.NewDecoder(r.Body).Decode(&strm)
	if err != nil {
		log.Error("Error decoding JSON response for ", channel.Name, " (", channel.ID, "): ", err)
	}

	gocache := viper.Get("Cache").(*cache.Cache)
	keyLive := "live:" + channel.Name

	if strm.Stream == nil {
		if _, ok := gocache.Get(keyLive); ok {
			<-throttle
			conn <- "PRIVMSG #" + channel.Name + " :" + channel.OfflineText
			gocache.Delete(keyLive)
		}
		return
	} else if strm.Stream.Channel.Name != channel.Name {
		channel.Name = strm.Stream.Channel.Name
	}

	if _, ok := gocache.Get(keyLive); !ok {
		<-throttle
		conn <- "PRIVMSG #" + channel.Name + " :" + channel.OnlineText
	}
	gocache.Set(keyLive, true, cache.DefaultExpiration)

	resp, err := tickClient.Get("http://tmi.twitch.tv/group/user/" + channel.Name + "/chatters")
	if err != nil {
		log.Error("Tick for ", channel.Name, " failed: ", err)
		return
	}
	defer resp.Body.Close()

	var chatters chatterList
	if err := json.NewDecoder(resp.Body).Decode(&chatters); err != nil {
		log.Error("Error decoding chatter JSON for ", channel.Name, ": ", err)
		return
	}

	if err := boltdb.Update(func(tx *bolt.Tx) error {
		for _, group := range chatters.Groups {
			for _, name := range group {
				b := tx.Bucket([]byte(channel.Name))
				donkeys := channel.GainRate
				buf := b.Get([]byte(name))
				if buf != nil {
					donkeys = binary.LittleEndian.Uint64(buf) + channel.GainRate
				}
				buf = make([]byte, 8)
				binary.LittleEndian.PutUint64(buf, donkeys)
				b.Put([]byte(name), buf)
			}
		}
		return nil
	}); err != nil {
		log.Error("Error updating donkey count for ", channel.Name, ": ", err)
	}

	now := time.Now()
	expiredCommand := false
	for _, cmd := range channel.Commands {
		if cmd.Expires.After(now) {
			db.Exec("DELETE FROM commands WHERE id = ?", cmd.ID)
			expiredCommand = true
		}
	}

	if expiredCommand {
		channel.NeedsUpdate <- true
	}
}
