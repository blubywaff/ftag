package backend

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"time"

	"github.com/blubywaff/ftag/internal/model"
	"github.com/google/uuid"
)

var TimeFormat = time.RFC3339
var TimeFormatP = []string{time.RFC3339, time.RFC3339Nano, "2006-01-02T15:04:05"}
var NoResult = errors.New("no data found")

func GenUUID() (string, error) {
	rid, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}
	id := rid.String()
	return id, nil
}

// Stores metadata for the system
// Including resource details, tags, and associations
type MetaStore interface {
	Connect(ctx context.Context) error
	AddResource(ctx context.Context, f io.Reader, tags model.TagSet) (string, error)
	GetResource(ctx context.Context, id string) (model.Resource, error)
	TagQuery(ctx context.Context, query model.Query) ([]model.Resource, error)
	ChangeTags(ctx context.Context, addtags model.TagSet, deltags model.TagSet, id string) error
	Close(ctx context.Context) error
}

type FileStore interface {
	Connect(ctx context.Context) error
	AddFile(ctx context.Context, f io.Reader) (string, error)
	GetFile(ctx context.Context, id string) (io.ReadCloser, error)
	Close(ctx context.Context) error
}
