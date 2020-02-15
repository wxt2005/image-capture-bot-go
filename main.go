package main

import (
	"crypto/tls"
	"net/http"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/wxt2005/image-capture-bot-go/controller"
	"github.com/wxt2005/image-capture-bot-go/db"
)

func init() {
	env := os.Getenv("ENV")
	if env == "DEBUG" || env == "LOCAL" {
		log.SetLevel(log.DebugLevel)
		// for debug, ignore ssl verify
		http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		log.Info("Disabled TLS certificate verify")
	}

	// init viper
	viper.SetConfigType("yaml")
	viper.SetConfigName("config")
	viper.AddConfigPath("./external")
	err := viper.ReadInConfig()
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Panic("failed to read config file")
	}

	// init db
	_, err = db.Init()
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Panic("failed to init bolt db")
	}
}

func main() {
	// get PORT from env, use with PORT=8080 or some
	port := os.Getenv("PORT")
	if len(port) == 0 {
		port = "8080"
	}

	http.HandleFunc("/api/"+viper.GetString("telegram.bot_token")+"/message", controller.MessageHandler)
	log.WithFields(log.Fields{
		"port": port,
	}).Info("Server listening...")
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.WithFields(log.Fields{
			"port": port,
		}).Panic("failed to listen port")
	}
}
