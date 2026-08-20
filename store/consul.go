package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/api/watch"
	"github.com/voc/rtmp-auth/storage"
	"google.golang.org/protobuf/proto"
)

type ConsulBackendConfig struct{}

const (
	consulStreamPrefix  = "stream/"
	consulStreamAuthKey = "/auth"
)

type ConsulBackend struct {
	cache     *storage.State
	client    *api.Client
	kv        *api.KV
	mutex     sync.RWMutex
	queryOpts api.QueryOptions
}

func NewConsulBackend(config ConsulBackendConfig) (Backend, error) {
	// Get a new client
	client, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		return nil, err
	}
	cb := &ConsulBackend{
		client:    client,
		kv:        client.KV(),
		queryOpts: api.QueryOptions{},
		cache:     &storage.State{},
	}

	_, err = cb.Read()
	if err != nil {
		return nil, err
	}

	err = cb.watch()
	if err != nil {
		return nil, fmt.Errorf("watch: %w", err)
	}

	return cb, nil
}

// Watch watches the consul stream keys for changes.
func (cb *ConsulBackend) watch() error {
	query := map[string]any{
		"type":   "keyprefix",
		"prefix": consulStreamPrefix,
	}
	plan, err := watch.Parse(query)
	if err != nil {
		return err
	}
	plan.HybridHandler = cb.handleWatch
	go func() {
		err := plan.RunWithClientAndHclog(cb.client, nil)
		log.Println("watch stopped", err)
	}()
	return nil
}

// handleWatch updates the cache on consul changes
func (cb *ConsulBackend) handleWatch(b watch.BlockingParamVal, update any) {
	pairs, ok := update.(api.KVPairs)
	if !ok {
		log.Println("watch: invalid update")
		return
	}
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	streams, err := streamsFromPairs(pairs)
	if err != nil {
		log.Printf("watch: parse streams: %v", err)
		return
	}
	cb.cache.Streams = streams
}

// Read directly from consul
func (cb *ConsulBackend) read() (*storage.State, error) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	pairs, _, err := cb.kv.List(consulStreamPrefix, cb.queryOpts.WithContext(ctx))
	if err != nil {
		return cb.getCache(), err
	}
	streams, err := streamsFromPairs(pairs)
	if err != nil {
		return cb.getCache(), fmt.Errorf("parse stream auth: %w", err)
	}
	cb.cache.Streams = streams
	return cb.getCache(), nil
}

func (cb *ConsulBackend) getCache() *storage.State {
	return cloneState(cb.cache)
}

// Read from backend
func (cb *ConsulBackend) Read() (*storage.State, error) {
	return cb.read()
}

// Write to consul KV
func (cb *ConsulBackend) Write(state *storage.State) error {
	if state == nil {
		return errors.New("state should not be nil")
	}

	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	pairs, _, err := cb.kv.List(consulStreamPrefix, cb.queryOpts.WithContext(ctx))
	if err != nil {
		return err
	}

	values, err := streamAuthValues(state.Streams)
	if err != nil {
		return err
	}
	operations := make([]*api.KVTxnOp, 0, len(values)+len(pairs))
	for key, value := range values {
		operations = append(operations, &api.KVTxnOp{Verb: api.KVSet, Key: key, Value: value})
	}
	for _, pair := range pairs {
		if _, exists := values[pair.Key]; !exists {
			operations = append(operations, &api.KVTxnOp{Verb: api.KVDelete, Key: pair.Key})
		}
	}

	_, _, _, err = cb.kv.Txn(operations, cb.queryOpts.WithContext(ctx))
	if err != nil {
		return err
	}
	cb.cache = cloneState(state)
	return nil
}

func streamAuthKey(name string) string {
	return consulStreamPrefix + name + consulStreamAuthKey
}

func streamAuthValues(streams []*storage.Stream) (map[string][]byte, error) {
	grouped := make(map[string][]*storage.Stream)
	for _, stream := range streams {
		grouped[streamAuthKey(stream.Name)] = append(grouped[streamAuthKey(stream.Name)], stream)
	}
	values := make(map[string][]byte, len(grouped))
	for key, entries := range grouped {
		value, err := json.Marshal(entries)
		if err != nil {
			return nil, fmt.Errorf("marshal %s: %w", key, err)
		}
		values[key] = value
	}
	return values, nil
}

func streamsFromPairs(pairs api.KVPairs) ([]*storage.Stream, error) {
	var streams []*storage.Stream
	for _, pair := range pairs {
		if !strings.HasSuffix(pair.Key, consulStreamAuthKey) {
			continue
		}
		var entries []*storage.Stream
		if err := json.Unmarshal(pair.Value, &entries); err != nil {
			return nil, fmt.Errorf("unmarshal %s: %w", pair.Key, err)
		}
		streams = append(streams, entries...)
	}
	return streams, nil
}

func cloneState(state *storage.State) *storage.State {
	return proto.Clone(state).(*storage.State)
}
