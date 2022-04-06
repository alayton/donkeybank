package commands

import (
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/alayton/donkeybank/chat/structs"
)

// Unmod allows the channel owner to remove moderator privileges for the bot from a user
func Unmod(channel *structs.ChannelSettings, msg structs.ServerMessage, conn chan string, throttle chan time.Time, rest string) {
	if msg.Source != channel.Name {
		return
	}

	parts := strings.SplitN(rest, " ", 2)
	toRemove := parts[0]

	removed := false
	for idx, name := range channel.Mods {
		if name == toRemove {
			channel.Mods = append(channel.Mods[:idx], channel.Mods[idx+1:]...)
			removed = true
		}
	}

	if !removed {
		return
	}

	db := viper.Get("Database").(*sqlx.DB)
	_, err := db.Exec("UPDATE channels SET mod_list = ? WHERE id = ?", strings.Join(channel.Mods, ","), channel.ID)
	if err != nil {
		log.Error("Error updating mod list: ", err)
		return
	}

	channel.NeedsUpdate <- true
}
