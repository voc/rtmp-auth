package store

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/api/watch"
	"github.com/voc/rtmp-auth/storage"
	"google.golang.org/protobuf/proto"
)

type ConsulBackendConfig struct{}

type ConsulBackend struct {
	cache     *storage.State
	client    *api.Client
	kv        *api.KV
	lastIndex uint64
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

	// Generate secret
	state, err := cb.Read()
	if err != nil {
		return nil, err
	}
	if len(state.Secret) == 0 {
		state.Secret = make([]byte, 32)
		rand.Read(state.Secret)
		err := cb.Write(state)
		if err != nil {
			return nil, err
		}
	}

	err = cb.watch()
	if err != nil {
		return nil, fmt.Errorf("watch: %w", err)
	}

	return cb, nil
}

// Watch watches the consul key for changes
func (cb *ConsulBackend) watch() error {
	query := map[string]interface{}{
		"type": "key",
		"key":  "stream_auth",
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
func (cb *ConsulBackend) handleWatch(b watch.BlockingParamVal, update interface{}) {
	pair, ok := update.(*api.KVPair)
	if !ok {
		log.Println("watch: invalid update")
		return
	}
	if pair == nil {
		return
	}
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	if err := proto.Unmarshal(pair.Value, cb.cache); err != nil {
		log.Println("watch: failed to parse state: %w", err)
	}
	cb.lastIndex = pair.ModifyIndex
}

// Read directly from consul
func (cb *ConsulBackend) read() (*storage.State, error) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	pair, _, err := cb.kv.Get("stream_auth", cb.queryOpts.WithContext(ctx))
	if err != nil {
		return cb.getCache(), err
	}

	if pair != nil {
		if err := proto.Unmarshal(pair.Value, cb.cache); err != nil {
			return cb.getCache(), fmt.Errorf("failed to parse state: %w", err)
		}
		cb.lastIndex = pair.ModifyIndex
	}

	return cb.getCache(), nil
}

func (cb *ConsulBackend) getCache() *storage.State {
	return proto.Clone(cb.cache).(*storage.State)
}

// Read from cache
func (cb *ConsulBackend) cachedRead() (*storage.State, error) {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	copy := proto.Clone(cb.cache).(*storage.State)
	return copy, nil
}

// Read from backend
func (cb *ConsulBackend) Read() (*storage.State, error) {
	if cb.lastIndex != 0 {
		return cb.cachedRead()
	}
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

	// marshal protobuf
	res, err := proto.Marshal(state)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	// put
	p := &api.KVPair{Key: "stream_auth", Value: res, ModifyIndex: cb.lastIndex}
	opts := api.WriteOptions{}
	success, _, err := cb.kv.CAS(p, opts.WithContext(ctx))
	if err != nil {
		return err
	}
	if !success {
		return errors.New("state changed during request, please try again")
	}
	// update directly so cached reads can return a correct response
	cb.cache = state

	return nil
}
