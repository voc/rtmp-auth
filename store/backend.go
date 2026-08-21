package store

import (
	"github.com/voc/rtmp-auth/storage"
)

type Backend interface {
	Read() (*storage.State, error)
	UpdateAuth(stream *storage.Stream) error
	RemoveAuth(stream *storage.Stream) error
}
