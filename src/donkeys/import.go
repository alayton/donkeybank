package main

import (
	"encoding/binary"
	"encoding/csv"
	"io"
	"os"
	"strconv"

	log "github.com/Sirupsen/logrus"
	"github.com/boltdb/bolt"
)

func importCsv(boltdb *bolt.DB) {
	// username, user id, current point, all time points
	file, err := os.Open("akamikeb_points.csv")
	if err != nil {
		log.Fatal("Opening file: ", err)
		return
	}

	r := csv.NewReader(file)

	boltdb.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("akamikeb"))

		for {
			row, err := r.Read()
			if err == io.EOF {
				break
			} else if err != nil {
				log.Fatal("CSV read: ", err)
				return nil
			}

			donkeys, err := strconv.ParseUint(row[2], 10, 64)
			if err != nil {
				log.Error("Error parsing donkeys: ", err)
				continue
			}

			buf := make([]byte, 8)
			binary.LittleEndian.PutUint64(buf, donkeys)
			b.Put([]byte(row[0]), buf)
		}

		return nil
	})
}
