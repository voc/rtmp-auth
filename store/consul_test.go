package store

import (
	"encoding/json"
	"testing"

	"github.com/hashicorp/consul/api"
	"github.com/voc/rtmp-auth/storage"
	"google.golang.org/protobuf/proto"
)

func TestStreamAuthValuesGroupsEntriesByName(t *testing.T) {
	values, err := streamAuthValues([]*storage.Stream{
		{Id: "first", Application: "live", Name: "camera", AuthKey: "one"},
		{Id: "second", Application: "backup", Name: "camera", AuthKey: "two"},
		{Id: "third", Application: "live", Name: "desk", AuthKey: "three"},
	})
	if err != nil {
		t.Fatalf("stream auth values: %v", err)
	}

	camera, ok := values["stream/camera/auth"]
	if !ok {
		t.Fatal("missing stream/camera/auth")
	}
	var entries []*storage.Stream
	if err := json.Unmarshal(camera, &entries); err != nil {
		t.Fatalf("decode camera auth: %v", err)
	}
	if len(entries) != 2 || entries[0].Id != "first" || entries[1].Id != "second" {
		t.Fatalf("camera entries = %#v, want first and second", entries)
	}
	if _, ok := values["stream/desk/auth"]; !ok {
		t.Fatal("missing stream/desk/auth")
	}
}

func TestStreamsFromPairsReadsAuthKeysOnly(t *testing.T) {
	streams, err := streamsFromPairs(api.KVPairs{
		{Key: "stream/camera/auth", Value: []byte(`[{"id":"first","name":"camera"}]`)},
		{Key: "stream/camera/metadata", Value: []byte(`ignored`)},
	})
	if err != nil {
		t.Fatalf("streams from pairs: %v", err)
	}
	if len(streams) != 1 || streams[0].Id != "first" {
		t.Fatalf("streams = %#v, want one camera auth entry", streams)
	}
}

func TestLegacyStreamAuthValues(t *testing.T) {
	legacy, err := proto.Marshal(&storage.State{Streams: []*storage.Stream{
		{Id: "first", Application: "live", Name: "camera", AuthKey: "one"},
		{Id: "second", Application: "live", Name: "camera", AuthKey: "two"},
	}})
	if err != nil {
		t.Fatalf("marshal legacy state: %v", err)
	}

	values, err := legacyStreamAuthValues(legacy)
	if err != nil {
		t.Fatalf("migrate legacy state: %v", err)
	}
	camera, ok := values["stream/camera/auth"]
	if !ok {
		t.Fatal("missing stream/camera/auth")
	}
	var entries []*storage.Stream
	if err := json.Unmarshal(camera, &entries); err != nil {
		t.Fatalf("decode camera auth: %v", err)
	}
	if len(entries) != 2 || entries[0].Id != "first" || entries[1].Id != "second" {
		t.Fatalf("camera entries = %#v, want first and second", entries)
	}
}
