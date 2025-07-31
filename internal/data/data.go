package data

import (
	"encoding/json"
	"errors"

	"github.com/blubywaff/ftag/internal/data/backend"
	"github.com/blubywaff/ftag/internal/data/filesystem"
	"github.com/blubywaff/ftag/internal/data/gremlin"
)

type Database = backend.MetaStore

type Config struct {
	FileStoreType string
	MetaStoreType string
	FileStore     json.RawMessage
	MetaStore     json.RawMessage
}

func Setup(config Config) (Database, error) {
	var fs backend.FileStore
	var ms backend.MetaStore
	var err error
	switch config.FileStoreType {
	case "filesystem":
		var conf filesystem.Config
		err = json.Unmarshal(config.FileStore, &conf)
		if err != nil {
			return nil, err
		}
		fs, err = filesystem.New(conf)
		if err != nil {
			return nil, err
		}
	default:
		return nil, errors.New("invalid FileStore type")
	}
	switch config.MetaStoreType {
	case "gremlin":
		var conf gremlin.Config
		err = json.Unmarshal(config.MetaStore, &conf)
		if err != nil {
			return nil, err
		}
		ms, err = gremlin.New(conf, fs)
	default:
		return nil, errors.New("invalid MetaStore type")
	}
	return ms, nil
}
