package main

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/boltdb/bolt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
	"github.com/justinas/alice"
	"github.com/patrickmn/go-cache"
	"github.com/spf13/viper"

	"donkeys/chat"
	"donkeys/respond"
)

var db *sqlx.DB
var boltdb *bolt.DB
var gocache *cache.Cache

func main() {
	setup()
	run()
}

func setup() {
	viper.SetConfigFile("config.yaml")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

	viper.SetDefault("Port", 3100)
	viper.SetDefault("Debug", false)

	err := viper.ReadInConfig()
	if err != nil {
		panic(err)
	}

	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:  true,
		DisableSorting: true,
		DisableColors:  true,
	})

	if viper.GetBool("Debug") {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}

	rand.Seed(time.Now().UTC().UnixNano())

	db = sqlx.MustConnect("mysql", fmt.Sprintf("%v:%v@tcp(%v)/%v?parseTime=true&collation=utf8mb4_unicode_ci", viper.Get("SqlUser"), viper.Get("SqlPass"), viper.Get("SqlHost"), viper.GetString("Database")))
	db = db.Unsafe()

	boltdb, err = bolt.Open("donkeys.db", 0600, nil)
	if err != nil {
		log.Fatal("Failed to open donkeys.db: ", err)
	}

	gocache = cache.New(40*time.Minute, 17*time.Minute)

	viper.Set("Database", db)
	viper.Set("Bolt", boltdb)
	viper.Set("Cache", gocache)
}

func liberalCORS(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Access-Control-Allow-Origin", "*")
		if r.Method == "OPTIONS" {
			w.Header().Add("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			if len(r.Header["Access-Control-Request-Headers"]) > 0 {
				w.Header().Add("Access-Control-Allow-Headers", r.Header.Get("Access-Control-Request-Headers"))
			}
			w.WriteHeader(200)
			return
		}

		h.ServeHTTP(w, r)
	})
}

func clientIP(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientIP := strings.TrimSpace(r.Header.Get("X-Real-Ip"))
		if len(clientIP) > 0 {
			r = r.WithContext(context.WithValue(r.Context(), "IP", clientIP))
			h.ServeHTTP(w, r)
			return
		}

		clientIP = r.Header.Get("X-Forwarded-For")
		if index := strings.IndexByte(clientIP, ','); index >= 0 {
			clientIP = strings.TrimSpace(clientIP[0:index])
		}
		if len(clientIP) > 0 {
			r = r.WithContext(context.WithValue(r.Context(), "IP", clientIP))
			h.ServeHTTP(w, r)
			return
		}

		r = r.WithContext(context.WithValue(r.Context(), "IP", r.RemoteAddr))
		h.ServeHTTP(w, r)
	})
}

func contextSetup(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r = r.WithContext(context.WithValue(r.Context(), "Database", db))
		h.ServeHTTP(w, r)
	})
}

func notFound(w http.ResponseWriter, r *http.Request) {
	respond.JSON(w, 404, struct {
		Error string
		Ok    bool
	}{
		"Not found",
		false,
	})
}

func run() {
	defer boltdb.Close()

	//importCsv(boltdb)
	//return

	r := mux.NewRouter()
	r.NotFoundHandler = http.HandlerFunc(notFound)

	go chat.Run(db)

	initRoutes(r)

	chain := alice.New(clientIP, reqTime, liberalCORS, recovery, contextSetup).Then(r)
	http.Handle("/", chain)

	log.Info("DonkeyBank is back in action!")
	http.ListenAndServe(":"+viper.GetString("Port"), nil)
}
