package store

import (
	"github.com/google/uuid"
	"github.com/voc/rtmp-auth/storage"
)

func ensureStreamID(stream *storage.Stream) error {
	if stream.Id != "" {
		return nil
	}
	id, err := uuid.NewUUID()
	if err != nil {
		return err
	}
	stream.Id = id.String()
	return nil
}

func replaceStreamAuth(existing []*storage.Stream, id string, replacement *storage.Stream) []*storage.Stream {
	streams := make([]*storage.Stream, 0, len(existing)+1)
	for _, stream := range existing {
		if stream.Id != id {
			streams = append(streams, stream)
		}
	}
	if replacement != nil {
		streams = append(streams, replacement)
	}
	return streams
}
