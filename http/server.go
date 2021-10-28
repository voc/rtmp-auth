package http

import (
	"context"
	"log"
	"sync"
	"time"

	"net/http"

	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
	"github.com/rakyll/statik/fs"
	_ "github.com/voc/rtmp-auth/statik"
	"github.com/voc/rtmp-auth/store"
)

type Frontend struct {
	server *http.Server
	done   sync.WaitGroup
}

func NewFrontend(address string, store *store.Store, prefix string, insecure bool) *Frontend {
	CSRF := csrf.Protect(store.State.Secret, csrf.Secure(!insecure))
	statikFS, err := fs.New()
	if err != nil {
		log.Fatal(err)
	}
	router := mux.NewRouter()
	sub := router.PathPrefix(prefix).Subrouter()
	sub.Path("/").Methods("GET").HandlerFunc(FormHandler(store))
	sub.Path("/add").Methods("POST").HandlerFunc(AddHandler(store))
	sub.Path("/remove").Methods("POST").HandlerFunc(RemoveHandler(store))
	sub.Path("/block").Methods("POST").HandlerFunc(BlockHandler(store))
	sub.PathPrefix("/public/").Handler(
		http.StripPrefix(prefix+"/public/", http.FileServer(statikFS)))

	frontend := &Frontend{
		server: &http.Server{
			Handler:      CSRF(router),
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

func NewAPI(address string, store *store.Store) *API {
	router := mux.NewRouter()
	router.Path("/publish").Methods("POST").HandlerFunc(PublishHandler(store))
	router.Path("/unpublish").Methods("POST").HandlerFunc(UnpublishHandler(store))

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
