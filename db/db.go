package db

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"go.etcd.io/bbolt"
)

var DB *bbolt.DB

func Init() (*bbolt.DB, error) {
	db, err := bbolt.Open(viper.GetString("db.db_path"), 0600, nil)
	if err != nil {
		return nil, err
	}
	var mainError error

	// create bucket
	db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(viper.GetString("db.url_bucket")))
		if err != nil {
			log.WithFields(log.Fields{
				"bucket": "url_bucket",
			}).Error("Failed to create bucket")
			mainError = err
			return err
		}

		_, err = tx.CreateBucketIfNotExists([]byte(viper.GetString("db.like_bucket")))
		if err != nil {
			log.WithFields(log.Fields{
				"bucket": "like_bucket",
			}).Error("Failed to create bucket")
			mainError = err
			return err
		}

		return nil
	})

	DB = db

	if mainError != nil {
		return nil, mainError
	}

	return DB, nil
}
