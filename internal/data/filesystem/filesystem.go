package filesystem

import (
	"context"
	"errors"
	"io"
)

type Config struct {
	Prefix string
}

type Filesystem struct {
    prefix string
}

func (fs *Filesystem) Connect(ctx context.Context) error {
	return errors.New("Not Implemented!")
}

func (fs *Filesystem) AddFile(ctx context.Context, f io.Reader) (string, error) {
	return "", errors.New("Not Implemented!")
}

func (fs *Filesystem) GetFile(ctx context.Context, id string) (io.ReadCloser, error) {
	return nil, errors.New("Not Implemented!")
}

func (fs *Filesystem) Close(ctx context.Context) error {
	return errors.New("Not Implemented!")
}

func New (config Config) (*Filesystem, error) {
    return &Filesystem{prefix: config.Prefix}, nil
}
