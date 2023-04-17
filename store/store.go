package store

import (
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"

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
		err = fmt.Errorf("Unknown backend %s", config.Backend)
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
		if stream.Application == app && stream.Name == name && stream.AuthKey == auth {
			if !stream.Blocked {
				var conflict bool
				if stream.Active {
					conflict = false
				} else {
					conflict = getAppNameActive(state, app, name)
				}
				return !conflict, stream.Id
			} else {
				return false, stream.Id
			}
		}
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
		if stream.Id == id {
			stream.Active = true
			if err := store.backend.Write(state); err != nil {
				log.Println(err)
			} else {
				success = true
			}
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
		if stream.Application == app && stream.Name == name {
			stream.Active = false
			if err := store.backend.Write(state); err != nil {
				log.Println(err)
			} else {
				success = true
			}
		}
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
			if err := store.backend.Write(state); err != nil {
				return err
			}
			return nil
		}
	}
	return nil
}

func (store *Store) AddStream(stream *storage.Stream) error {
	id, err := uuid.NewUUID()
	if err != nil {
		return err
	}

	stream.Id = id.String()
	stream.Blocked = false
	state, err := store.backend.Read()
	if err != nil {
		return err
	}
	state.Streams = append(state.Streams, stream)

	if err := store.backend.Write(state); err != nil {
		return err
	}

	return nil
}

func (store *Store) RemoveStream(id string) error {
	state, err := store.backend.Read()
	if err != nil {
		return err
	}
	s := state.Streams
	found := false
	var index int
	var stream *storage.Stream
	for index, stream = range s {
		if stream.Id == id {
			found = true
			break
		}
	}

	if found {
		copy(s[index:], s[index+1:]) // Shift a[i+1:] left one index
		s[len(s)-1] = nil            // Erase last element (write zero value)
		state.Streams = s[:len(s)-1] // Truncate slice
	}

	if err := store.backend.Write(state); err != nil {
		return err
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
		if stream.AuthExpire != -1 && stream.AuthExpire < now {
			log.Printf("Expiring %s/%s\n", stream.Application, stream.Name)
			toDelete = append(toDelete, stream.Id)
		}
	}

	for _, id := range toDelete {
		store.RemoveStream(id)
	}
}

func (store *Store) Get() (*storage.State, error) {
	return store.backend.Read()
}
