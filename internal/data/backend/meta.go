package backend

import (
	"context"
	"crypto/sha512"
	"encoding/base64"
	"errors"
	"hash"
	"io"
	"net/http"
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

type FileIdentity struct {
	Mime string
	Sum string
}

type fileIdentifier struct {
	origin io.Reader
	bts []byte
	sat int
	hasher hash.Hash
	Channel chan FileIdentity
	err error
}

func (fi *fileIdentifier) Read(p []byte) (int, error) {
	if fi.err != nil {
		return 0, fi.err
	}
	n, err := fi.origin.Read(p)
	if fi.sat < len(fi.bts) {
		copy(fi.bts[fi.sat:], p)
		fi.sat += n
	}
	if err == io.EOF {
		go func() {
			fi.Channel <- FileIdentity{
				Mime: http.DetectContentType(fi.bts),
				Sum: base64.StdEncoding.EncodeToString(fi.hasher.Sum([]byte{})),
			}
			close(fi.Channel)
		} ()
	} else if err != nil {
		close(fi.Channel)
	}
	fi.err = err
	return n, err
}

func newFileIdentifier(f io.Reader) *fileIdentifier {
	h := sha512.New()
	return &fileIdentifier{
		hasher: h,
		origin: io.TeeReader(f, h),
		sat: 0,
		bts: make([]byte, 512),
		Channel: make(chan FileIdentity),
		err: nil,
	}
}

// The caller MUST read from the channel, with the possibility of it being closed
func IdentifyFile(f io.Reader) (io.Reader, chan FileIdentity, error) {
	fi := newFileIdentifier(f)
	return fi, fi.Channel, nil
}

// Stores metadata for the system
// Including resource details, tags, and associations
type MetaStore interface {
	// Setup
	Connect(ctx context.Context) error

	// Create
	AddResource(ctx context.Context, f io.Reader, tags model.TagSet) (string, error)

	// Read
	GetResource(ctx context.Context, id string) (model.Resource, error)
	TagQuery(ctx context.Context, query model.Query) ([]model.Resource, error)
	GetFile(ctx context.Context, id string) (io.ReadCloser, error)
	GetTags(ctx context.Context) (model.TagSet, error)

	// Update
	ChangeTags(ctx context.Context, addtags model.TagSet, deltags model.TagSet, id string) error
	
	// Delete (None)

	// Shutdown
	Close(ctx context.Context) error
}

type FileStore interface {
	Connect(ctx context.Context) error
	AddFile(ctx context.Context, id string, f io.Reader) (error)
	GetFile(ctx context.Context, id string) (io.ReadCloser, error)
	Close(ctx context.Context) error
}
