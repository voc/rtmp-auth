package store

import (
	"fmt"
	"log"
	"time"

	"github.com/voc/rtmp-auth/storage"
)

type StoreConfig struct {
	Backend string
	File    FileBackendConfig
	Consul  ConsulBackendConfig
}

type Store struct {
	backend Backend
}

func NewStore(config StoreConfig) (*Store, error) {
	var backend Backend
	var err error
	switch config.Backend {
	case "file":
		backend, err = NewFileBackend(config.File)
	case "consul":
		backend, err = NewConsulBackend(config.Consul)
	default:
		err = fmt.Errorf("unknown backend %s", config.Backend)
	}
	if err != nil {
		return nil, err
	}
	log.Printf("store: using %s backend\n", config.Backend)
	return &Store{backend: backend}, nil
}

// GetAppNameActive returns true if there is an active stream on app/name
func getAppNameActive(state *storage.State, app string, name string) bool {
	active := false
	for _, stream := range state.Streams {
		if stream.Application == app && stream.Name == name && stream.Active {
			active = true
		}
	}
	return active
}

// Auth looks up if a given app/name/key tuple is allowed to publish.
// Returns success (bool) and the matched streams id string
// TODO: Return error values to distinguish i.e. 401 Unauthorized
// and 409 Conflict return codes in the publish request handler
func (store *Store) Auth(app string, name string, auth string) (success bool, id string) {
	state, err := store.backend.Read()
	if err != nil {
		return false, ""
	}

	for _, stream := range state.Streams {
		if (stream.Application != "" && stream.Application != app) || stream.Name != name || stream.AuthKey != auth {
			continue
		}
		if stream.Blocked {
			return false, stream.Id
		}
		if !stream.Active && getAppNameActive(state, app, name) {
			return false, stream.Id
		}
		return true, stream.Id
	}
	return false, ""
}

// SetActive sets a stream to active state by its id, returns success
func (store *Store) SetActive(id string) bool {
	state, err := store.backend.Read()
	if err != nil {
		return false
	}

	success := false
	for _, stream := range state.Streams {
		if stream.Id != id {
			continue
		}
		stream.Active = true
		if err := store.backend.UpdateAuth(stream); err != nil {
			log.Println(err)
		} else {
			success = true
		}
	}
	return success
}

// SetInactive unsets the active state for all streams defined for app/name, returns success
func (store *Store) SetInactive(app string, name string) bool {
	state, err := store.backend.Read()
	if err != nil {
		return false
	}

	success := false
	for _, stream := range state.Streams {
		if stream.Application != app || stream.Name != name {
			continue
		}
		stream.Active = false
		if err := store.backend.UpdateAuth(stream); err != nil {
			log.Println(err)
			return false
		}
		success = true
	}
	return success
}

// SetBlocked changes a streams blocked state
func (store *Store) SetBlocked(id string, isBlocked bool) error {
	state, err := store.backend.Read()
	if err != nil {
		return err
	}

	for _, stream := range state.Streams {
		if stream.Id == id {
			stream.Blocked = isBlocked
			if err := store.backend.UpdateAuth(stream); err != nil {
				return err
			}
			return nil
		}
	}
	return nil
}

func (store *Store) AddStream(stream *storage.Stream) error {
	if err := ensureStreamID(stream); err != nil {
		return err
	}
	if err := store.backend.UpdateAuth(stream); err != nil {
		return err
	}

	return nil
}

func (store *Store) RemoveStream(id string) error {
	state, err := store.backend.Read()
	if err != nil {
		return err
	}
	for _, stream := range state.Streams {
		if stream.Id != id {
			continue
		}
		if err := store.backend.RemoveAuth(stream); err != nil {
			return err
		}
		break
	}
	return nil
}

// Expire old streams
func (store *Store) Expire() {
	var toDelete []string
	now := time.Now().Unix()

	state, err := store.backend.Read()
	if err != nil {
		log.Println("read", err)
	}

	for _, stream := range state.Streams {
		if stream.AuthExpire != -1 && stream.AuthExpire != 0 && stream.AuthExpire < now {
			log.Printf("Expiring %s/%s\n", stream.Application, stream.Name)
			toDelete = append(toDelete, stream.Id)
		}
	}

	for _, id := range toDelete {
		if err := store.RemoveStream(id); err != nil {
			log.Println("remove stream", err)
		}
	}
}

func (store *Store) Get() (*storage.State, error) {
	return store.backend.Read()
}
