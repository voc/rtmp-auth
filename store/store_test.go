package store

import (
	"testing"

	"github.com/voc/rtmp-auth/storage"
	"google.golang.org/protobuf/proto"
)

type recordingBackend struct {
	state       *storage.State
	updatedName string
	updated     []*storage.Stream
	removedName string
}

func (backend *recordingBackend) Read() (*storage.State, error) {
	return proto.Clone(backend.state).(*storage.State), nil
}

func (backend *recordingBackend) UpdateAuth(stream *storage.Stream) error {
	backend.updatedName = stream.Name
	backend.updated = []*storage.Stream{stream}
	backend.state.Streams = replaceStreamAuth(backend.state.Streams, stream.Id, stream)
	return nil
}

func (backend *recordingBackend) RemoveAuth(stream *storage.Stream) error {
	backend.removedName = stream.Name
	backend.state.Streams = replaceStreamAuth(backend.state.Streams, stream.Id, nil)
	return nil
}

func TestRemoveStreamRemovesEmptyStreamAuth(t *testing.T) {
	backend := &recordingBackend{state: &storage.State{Streams: []*storage.Stream{
		{Id: "camera", Name: "camera"},
		{Id: "desk", Name: "desk"},
	}}}
	streamStore := &Store{backend: backend}

	if err := streamStore.RemoveStream("camera"); err != nil {
		t.Fatalf("remove stream: %v", err)
	}
	if backend.removedName != "camera" {
		t.Fatalf("removed name = %q, want camera", backend.removedName)
	}
	if backend.updated != nil {
		t.Fatalf("updated auth = %#v, want none", backend.updated)
	}
}
