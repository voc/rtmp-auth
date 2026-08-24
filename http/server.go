package http

import (
	"context"
	"log"
	"sync"
	"time"

	"net/http"

	"github.com/voc/rtmp-auth/public"
	"github.com/voc/rtmp-auth/store"
)

type ServerConfig struct {
	Applications []string `toml:"applications"`
	Prefix       string   `toml:"prefix"`
}

type Frontend struct {
	server *http.Server
	done   sync.WaitGroup
}

func NewFrontend(address string, config ServerConfig, store *store.Store) *Frontend {
	router := http.NewServeMux()
	router.HandleFunc("GET "+config.Prefix+"/{$}", FormHandler(store, config))
	router.HandleFunc("POST "+config.Prefix+"/add", AddHandler(store, config))
	router.HandleFunc("POST "+config.Prefix+"/remove", RemoveHandler(store, config))
	router.HandleFunc("POST "+config.Prefix+"/block", BlockHandler(store, config))
	router.Handle("GET "+config.Prefix+"/public/", http.StripPrefix(config.Prefix+"/public/", http.FileServer(http.FS(public.FS))))
	handler := http.NewCrossOriginProtection().Handler(router)

	frontend := &Frontend{
		server: &http.Server{
			Handler:      handler,
			Addr:         address,
			WriteTimeout: 15 * time.Second,
			ReadTimeout:  15 * time.Second,
		},
	}

	frontend.done.Add(1)
	go func() {
		defer frontend.done.Done()
		log.Println("Frontend Listening on", frontend.server.Addr)
		if err := frontend.server.ListenAndServe(); err != http.ErrServerClosed {
			log.Println(err)
		}
	}()
	return frontend
}

func (frontend *Frontend) Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*500)
	defer cancel()
	if err := frontend.server.Shutdown(ctx); err != nil {
		log.Println("frontend shutdown:", err)
	}
	frontend.done.Wait()
}

type API struct {
	server *http.Server
	done   sync.WaitGroup
}

func NewAPI(address string, config ServerConfig, store *store.Store) *API {
	router := http.NewServeMux()
	router.HandleFunc("POST /publish", PublishHandler(store))
	router.HandleFunc("POST /unpublish", UnpublishHandler(store))
	router.HandleFunc("POST /auth", AuthHandler(store))

	api := &API{
		server: &http.Server{
			Handler:      router,
			Addr:         address,
			WriteTimeout: 15 * time.Second,
			ReadTimeout:  15 * time.Second,
		},
	}

	api.done.Add(1)
	go func() {
		defer api.done.Done()
		log.Println("API Listening on", api.server.Addr)
		if err := api.server.ListenAndServe(); err != http.ErrServerClosed {
			log.Println(err)
		}
	}()

	return api
}

func (api *API) Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*500)
	defer cancel()
	if err := api.server.Shutdown(ctx); err != nil {
		log.Println("api shutdown:", err)
	}
	api.done.Wait()
}
