package http

import (
	"context"
	"crypto/rand"
	"log"
	"sync"
	"time"

	"net/http"

	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
	"github.com/voc/rtmp-auth/public"
	"github.com/voc/rtmp-auth/store"
)

type ServerConfig struct {
	Applications []string `toml:"applications"`
	Prefix       string   `toml:"prefix"`
	Insecure     bool     `toml:"insecure"`
}

type Frontend struct {
	server *http.Server
	done   sync.WaitGroup
}

func NewFrontend(address string, config ServerConfig, store *store.Store) *Frontend {
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		log.Fatal("generate CSRF secret", err)
	}

	router := mux.NewRouter()
	sub := router.PathPrefix(config.Prefix).Subrouter()
	sub.Path("/").Methods("GET").HandlerFunc(FormHandler(store, config))
	sub.Path("/add").Methods("POST").HandlerFunc(AddHandler(store, config))
	sub.Path("/remove").Methods("POST").HandlerFunc(RemoveHandler(store, config))
	sub.Path("/block").Methods("POST").HandlerFunc(BlockHandler(store, config))
	sub.PathPrefix("/public/").Handler(
		http.StripPrefix(config.Prefix+"/public/", http.FileServer(http.FS(public.FS))))
	CSRF := csrf.Protect(secret, csrf.Secure(!config.Insecure))
	csrfHandler := CSRF(router)
	var handler http.Handler = csrfHandler
	if config.Insecure {
		handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			csrfHandler.ServeHTTP(w, csrf.PlaintextHTTPRequest(r))
		})
	}

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
	router := mux.NewRouter()
	router.Path("/publish").Methods("POST").HandlerFunc(PublishHandler(store))
	router.Path("/unpublish").Methods("POST").HandlerFunc(UnpublishHandler(store))
	router.Path("/auth").Methods("POST").HandlerFunc(AuthHandler(store))

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
