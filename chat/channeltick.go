package chat

import (
	"encoding/binary"
	"encoding/json"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/boltdb/bolt"
	"github.com/jmoiron/sqlx"
	"github.com/patrickmn/go-cache"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/alayton/donkeybank/chat/structs"
)

type streamStatus struct {
	Data []struct {
		ID     string `json:"id"`
		UserID string `json:"user_id"`
		Name   string `json:"user_name"`
		Game   string `json:"game_name"`
		Type   string `json:"type"`
	} `json:"data"`
}

type authResult struct {
	Token string `json:"access_token"`
}

var tickClient = &http.Client{Timeout: 15 * time.Second}
var authMutex sync.RWMutex
var authToken string

type chatterList struct {
	Count  int                 `json:"chatter_count"`
	Groups map[string][]string `json:"chatters"`
}

func channelTick(db *sqlx.DB, boltdb *bolt.DB, channel *structs.ChannelSettings, conn chan string, throttle chan time.Time) {
	clientID := viper.GetString("TwitchClientID")
	clientSecret := viper.GetString("TwitchClientSecret")

	req, err := http.NewRequest("GET", "https://api.twitch.tv/helix/streams?user_id="+strconv.FormatInt(channel.ID, 10), nil)
	if err != nil {
		log.Error("Error stream status request for ", channel.Name, ": ", err)
		return
	}
	req.Header.Add("Client-Id", clientID)

	authMutex.RLock()
	req.Header.Add("Authorization", "Bearer "+authToken)
	authMutex.RUnlock()
	r, err := tickClient.Do(req)
	if err != nil {
		log.Error("Error fetching stream status for ", channel.Name, " (", channel.ID, "): ", err)
		return
	}
	defer r.Body.Close()

	if r.StatusCode == 401 {
		authMutex.Lock()
		defer authMutex.Unlock()

		auth, err := tickClient.Post("https://id.twitch.tv/oauth2/token?client_id="+clientID+"&client_secret="+clientSecret+"&grant_type=client_credentials", "application/x-www-form-urlencoded", nil)
		if err != nil {
			log.Error("Error getting auth token: ", err)
			return
		}
		defer auth.Body.Close()

		var authBody authResult
		err = json.NewDecoder(auth.Body).Decode(&authBody)
		if err != nil {
			log.Error("Error decoding auth JSON response")
			return
		}

		if len(authBody.Token) == 0 {
			log.Error("Got an empty auth token")
			return
		}

		authToken = authBody.Token
		return
	}

	var strm streamStatus
	err = json.NewDecoder(r.Body).Decode(&strm)
	if err != nil {
		log.Error("Error decoding JSON response for ", channel.Name, " (", channel.ID, "): ", err)
	}

	gocache := viper.Get("Cache").(*cache.Cache)
	keyLive := "live:" + channel.Name

	if len(strm.Data) == 0 || strm.Data[0].Type != "live" {
		if _, ok := gocache.Get(keyLive); ok {
			<-throttle
			conn <- "PRIVMSG #" + channel.Name + " :" + channel.OfflineText
			gocache.Delete(keyLive)
		}
		return
	} else if strm.Data[0].Name != channel.Name {
		channel.Name = strm.Data[0].Name
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
