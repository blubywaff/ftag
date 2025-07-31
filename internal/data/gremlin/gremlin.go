package gremlin

import (
	"context"
	"errors"
	"io"
	"time"

	gg "github.com/apache/tinkerpop/gremlin-go/v3/driver"
	"github.com/blubywaff/ftag/internal/data/backend"
	"github.com/blubywaff/ftag/internal/model"
)

type Gremlin struct {
	ts     *TraversalSource
	fs     backend.FileStore
	remote *gg.DriverRemoteConnection
}

func (g *Gremlin) Connect(ctx context.Context) error {
	g.ts = &TraversalSource{gg.Traversal_().WithRemote(g.remote)}
	g.fs.Connect(ctx)
	return nil
}

func (g *Gremlin) AddResource(ctx context.Context, f io.Reader, tags model.TagSet) (string, error) {
	var rsc model.Resource
	id, err := backend.GenUUID()
	if err != nil {
		return "", err
	}
	rsc.Id = id
	rsc.CreatedAt = time.Now()
	rsc.Tags = tags
	fi, fic, err := backend.IdentifyFile(f)
	if err != nil {
		return "", err
	}
	err = g.fs.AddFile(ctx, id, fi)
	rid, ok := <- fic
	if !ok {
		return "", errors.New("Could not extract file information")
	}
	rsc.Mimetype = rid.Mime
	err = <- g.ts.createResource(rsc).Iterate()
	if err != nil {
		return "", err
	}
	return "", nil
}

func (g *Gremlin) GetResource(ctx context.Context, id string) (model.Resource, error) {
	rsc, err := g.ts.resources().withId(id).toResource()
	if len(rsc) == 0 {
		return model.Resource{}, backend.NoResult
	}
	return rsc[0], err
}

func (g *Gremlin) TagQuery(ctx context.Context, query model.Query) ([]model.Resource, error) {
	rsc, err := g.ts.resources().withTags(query.Include).withoutTags(query.Exclude).sort().part(query.Offset, query.Limit).toResource()
	return rsc, err
}

func (g *Gremlin) GetTags(ctx context.Context) (model.TagSet, error) {
	ts, err := g.ts.tags().toTags()
	if err != nil {
		return model.TagSet{}, nil
	}
	return ts, nil
}

func (g *Gremlin) GetFile(ctx context.Context, id string) (io.ReadCloser, error) {
	return g.fs.GetFile(ctx, id)
}

func (g *Gremlin) ChangeTags(ctx context.Context, addtags model.TagSet, deltags model.TagSet, id string) error {
	tx, s, err := g.ts.tx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	err = s.resources().withId(id).addTags(addtags)
	if err != nil {
		return err
	}
	err = s.resources().withId(id).removeTags(deltags)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func (g *Gremlin) Close(ctx context.Context) error {
	g.fs.Close(ctx)
	g.remote.Close()
	return nil
}

func New(config Config, fs backend.FileStore) (*Gremlin, error) {
	remote, err := gg.NewDriverRemoteConnection(config.Url)
	if err != nil {
		return nil, err
	}
	return &Gremlin{fs: fs, remote: remote}, nil
}
