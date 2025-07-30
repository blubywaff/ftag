package filesystem

import (
	"context"
	"io"
	"os"
)

type Config struct {
	Prefix string
}

type Filesystem struct {
	prefix string
}

func (fs *Filesystem) Connect(ctx context.Context) error {
	// No connect necessary for FS
	return nil
}

func (fs *Filesystem) AddFile(ctx context.Context, id string, f io.Reader) error {
	file, err := os.OpenFile(fs.prefix+id, os.O_WRONLY|os.O_CREATE, os.ModePerm)
	if err != nil {
		return err
	}

	_, err = io.Copy(file, f)
	return err
}

func (fs *Filesystem) GetFile(ctx context.Context, id string) (io.ReadCloser, error) {
	file, err := os.OpenFile(fs.prefix+id, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return nil, err
	}

	return file, nil
}

func (fs *Filesystem) Close(ctx context.Context) error {
	// No disconnect necessary for FS
	return nil
}

func New(config Config) (*Filesystem, error) {
	return &Filesystem{prefix: config.Prefix}, nil
}
