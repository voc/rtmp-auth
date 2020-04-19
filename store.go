package main

import (
	"os"
	"fmt"
	"log"
	"time"
	"sync"
	"io/ioutil"
	"github.com/google/uuid"
	"github.com/golang/protobuf/proto"

	"github.com/voc/rtmp-auth/storage"
)

type Store struct {
	State storage.State
	Applications []string
	Path string
	sync.RWMutex
}

func NewStore(path string, apps []string) (*Store, error) {
	store := &Store{Path: path, Applications: apps}
	if err := store.read(); err != nil {
			return nil, err
	}

	return store, nil
}

func (store *Store) Auth(app string, name string, auth string) bool {
	store.RLock()
	defer store.RUnlock()
	for _, stream := range store.State.Streams {
		if stream.Application == app && stream.Name == name && stream.AuthKey == auth {
			return true
		}
	}

	return false
}

// SetActive changes a streams active state, returns success
func (store *Store) SetActive(app string, name string, state bool) bool {
	store.Lock()
	defer store.Unlock()
	for _, stream := range store.State.Streams {
		if stream.Application == app && stream.Name == name {
			stream.Active = state
			if err := store.save(); err != nil{
				log.Println(err)
			}
			return true
		}
	}

	return false
}

func (store *Store) AddStream(stream *storage.Stream) error {
	store.Lock()
	defer store.Unlock()

	id, err := uuid.NewUUID()
	if err != nil {
		return err
	}

	stream.Id = id.String();
	store.State.Streams = append(store.State.Streams, stream)

	if err := store.save(); err != nil{
		return err
	}

	return nil
}

func (store *Store) RemoveStream(id string) error {
	store.Lock()
	defer store.Unlock()

	s := store.State.Streams
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
		copy(s[index:], s[index+1:]) 		// Shift a[i+1:] left one index
		s[len(s)-1] = nil     				// Erase last element (write zero value)
		store.State.Streams = s[:len(s)-1]  // Truncate slice
	}

	if err := store.save(); err != nil{
		return err
	}

	return nil
}

// Expire old streams
func (store *Store) Expire() {
	var toDelete []string
	now := time.Now().Unix()

	store.RLock()
	for _, stream := range store.State.Streams {
		if stream.AuthExpire != -1 && stream.AuthExpire < now {
			log.Printf("Expiring %s/%s\n", stream.Application, stream.Name)
			toDelete = append(toDelete, stream.Id)
		}
	}
	store.RUnlock()

	for _, id := range toDelete{
		store.RemoveStream(id)
	}
}

func (store *Store) Get() Store {
	store.RLock()
	defer store.RUnlock()
	return *store
}

// Read parses the store state from a file
func (store *Store) read() error {
	store.Lock()
	defer store.Unlock()
	data, err := ioutil.ReadFile(store.Path)
	if err != nil {
			// Non-existing state is ok
			if os.IsNotExist(err) {
					return nil
			}
			return fmt.Errorf("No previous file read: %v", err)
	}
	if err := proto.Unmarshal(data, &store.State); err != nil {
			return fmt.Errorf("Failed to parse stream state: %v", err)
	}
	log.Println("State restored from", store.Path)
	return nil
}

// Save stores the store state in a file
// Requires Lock
func (store *Store) save() error {
	out, err := proto.Marshal(&store.State)
	if err != nil {
			return fmt.Errorf("Failed to encode state: %v", err)
	}
	tmp := fmt.Sprintf(store.Path + ".%v", time.Now())
	if err := ioutil.WriteFile(tmp, out, 0600); err != nil {
			return fmt.Errorf("Failed to write state: %v", err)
	}
	err = os.Rename(tmp, store.Path)
	if err != nil {
		return fmt.Errorf("Failed to move state: %v", err)
	}
	return nil
}
