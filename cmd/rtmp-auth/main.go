package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pelletier/go-toml"
	"github.com/voc/rtmp-auth/http"
	"github.com/voc/rtmp-auth/store"
)

func waitForSignal() {
	// Set up channel on which to send signal notifications.
	// We must use a buffered channel or risk missing the signal
	// if we're not ready to receive when the signal is sent.
	c := make(chan os.Signal, 1)
	done := make(chan struct{})
	signal.Notify(c,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM)

	go func() {
		for s := range c {
			log.Println("caught signal", s)
			if s == syscall.SIGHUP {
				continue
			}
			close(done)
			return
		}
	}()
	<-done
}

type Config struct {
	APIAddress      string            `toml:"api-address"`
	FrontendAddress string            `toml:"frontend-address"`
	Store           store.StoreConfig `toml:"store"`
	HTTP            http.ServerConfig `toml:"http"`
}

func main() {
	// default config
	config := Config{
		APIAddress:      "localhost:8080",
		FrontendAddress: "localhost:8082",
		Store: store.StoreConfig{
			Backend: "file",
			File: store.FileBackendConfig{
				Path: "store.db",
			},
		},
	}
	var configPath = flag.String("config", "config.toml", "Config toml")
	var apiAddr = flag.String("apiAddr", "", "API bind address")
	var frontendAddr = flag.String("frontendAddr", "", "Frontend bind address")
	var insecure = flag.Bool("insecure", false, "Set to allow non-secure CSRF cookie")
	var prefix = flag.String("subpath", "", "Set to allow running behind reverse-proxy at that subpath")
	flag.Parse()

	if *apiAddr != "" {
		config.APIAddress = *apiAddr
	}
	if *frontendAddr != "" {
		config.FrontendAddress = *frontendAddr
	}
	if *insecure {
		config.HTTP.Insecure = true
	}
	if *prefix != "" {
		config.HTTP.Prefix = *prefix
	}

	res, err := ioutil.ReadFile(*configPath)
	if err != nil {
		log.Fatal("read config", err)
	}
	err = toml.Unmarshal(res, &config)
	if err != nil {
		log.Fatal("parse config", err)
	}

	out, _ := json.Marshal(&config)
	log.Println("using config", string(out))

	store, err := store.NewStore(config.Store)
	if err != nil {
		log.Fatal("Failed to create store", err)
	}

	// Set up servers
	api := http.NewAPI(config.APIAddress, config.HTTP, store)
	frontend := http.NewFrontend(config.FrontendAddress, config.HTTP, store)

	// Periodically expire old streams
	ticker := time.NewTicker(5 * time.Minute)
	stopPolling := make(chan struct{})
	go func() {
		for {
			select {
			case <-stopPolling:
				return
			case <-ticker.C:
				store.Expire()
			}
		}
	}()

	// Handle signals
	waitForSignal()
	log.Println("Shutting down")

	// Shut everything down
	close(stopPolling)
	api.Stop()
	frontend.Stop()
}
