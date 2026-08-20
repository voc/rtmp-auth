package main

import (
	"log"

	"github.com/hashicorp/consul/api"
	"github.com/voc/rtmp-auth/store"
)

func main() {
	client, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		log.Fatal("create Consul client:", err)
	}
	migrated, err := store.MigrateConsul(client)
	if err != nil {
		log.Fatal("migrate Consul state:", err)
	}
	if !migrated {
		log.Println("no legacy Consul state found")
		return
	}
	log.Println("migrated legacy Consul state")
}