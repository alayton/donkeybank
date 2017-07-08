package commands

import (
	"regexp"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/jmoiron/sqlx"
	"github.com/spf13/viper"

	"donkeys/chat/structs"
)

func isMod(channel *structs.ChannelSettings, msg structs.ServerMessage) bool {
	for _, name := range channel.Mods {
		if name == strings.ToLower(msg.Source) {
			return true
		}
	}
	return false
}

// Mod allows the channel owner to give a user moderator privileges for the bot
func Mod(channel *structs.ChannelSettings, msg structs.ServerMessage, conn chan string, throttle chan time.Time, rest string) {
	if msg.Source != channel.Name {
		return
	}

	parts := strings.SplitN(rest, " ", 2)
	newMod := parts[0]
	matches, err := regexp.MatchString("^[a-zA-Z0-9_]{4,25}$", newMod)
	if err != nil {
		log.Error("Error testing mod name: ", err)
		return
	} else if !matches {
		return
	}

	for _, name := range channel.Mods {
		if name == newMod {
			return
		}
	}

	mods := append(channel.Mods, newMod)
	db := viper.Get("Database").(*sqlx.DB)
	_, err = db.Exec("UPDATE channels SET mod_list = ? WHERE id = ?", strings.Join(mods, ","), channel.ID)
	if err != nil {
		log.Error("Error updating mod list: ", err)
		return
	}

	channel.NeedsUpdate <- true
}
