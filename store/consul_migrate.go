package store

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/voc/rtmp-auth/storage"
	"google.golang.org/protobuf/proto"
)

const legacyConsulStateKey = "stream_auth"

// MigrateConsul migrates the legacy stream_auth protobuf value to stream auth keys.
// It returns false when no legacy state exists.
func MigrateConsul(client *api.Client) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	legacy, _, err := client.KV().Get(legacyConsulStateKey, (&api.QueryOptions{}).WithContext(ctx))
	if err != nil {
		return false, fmt.Errorf("read legacy state: %w", err)
	}
	if legacy == nil {
		return false, nil
	}
	values, err := legacyStreamAuthValues(legacy.Value)
	if err != nil {
		return false, err
	}
	for key, value := range values {
		if _, err := client.KV().Put(&api.KVPair{Key: key, Value: value}, nil); err != nil {
			return false, fmt.Errorf("write %s: %w", key, err)
		}
	}
	if _, err := client.KV().Delete(legacyConsulStateKey, nil); err != nil {
		return false, fmt.Errorf("delete legacy state: %w", err)
	}
	return true, nil
}

func legacyStreamAuthValues(data []byte) (map[string][]byte, error) {
	state := &storage.State{}
	if err := proto.Unmarshal(data, state); err != nil {
		return nil, fmt.Errorf("parse legacy state: %w", err)
	}
	return streamAuthValues(state.Streams)
}
