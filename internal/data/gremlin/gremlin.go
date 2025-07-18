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

var g_ = gg.T__
var gtp = gg.TextP

var gp = gg.P
var gs = gg.Order
var gc = gg.Column
var gm = gg.Merge
var gt = gg.T
var gd = gg.Direction

func ToInterfaceSlice[T any](in []T) []interface{} {
	res := make([]interface{}, len(in))
	for i, v := range in {
		res[i] = v
	}
	return res
}

func FromInterfaceSlice[T any](in []interface{}) ([]T, error) {
	res := make([]T, len(in))
	var ok bool
	for i, v := range in {
		res[i], ok = v.(T)
		if !ok {
			return nil, errors.New("invalid type within slice conversion")
		}
	}
	return res, nil
}

type Config struct {
	Url string
}

type TraversalSource struct {
	*gg.GraphTraversalSource
}

func (s *TraversalSource) tx() (*gg.Transaction, *TraversalSource, error) {
    tx := s.Tx()
    ts, err := tx.Begin()
    return tx, &TraversalSource{ts}, err
}

func (s *TraversalSource) createResource(rsc model.Resource) *Traversal {
    return &Traversal{
        s.AddV("resource").
        Property("rsc_id", rsc.Id).
        Property("upload", rsc.CreatedAt).
        Property("rsc_id", rsc.Mimetype),
    }
}

func (s *TraversalSource) tags() *Traversal {
    return &Traversal{
        s.V().HasLabel("tag"),
    }
}

func (s *TraversalSource) resources() *Traversal {
    t := s.GetGraphTraversal()
    return &Traversal{
        t.V().HasLabel("resource"),
    }
}

type Traversal struct {
	*gg.GraphTraversal
}

type iAnonymousTraversal interface {
    gg.AnonymousTraversal
}

type AnonymousTraversal struct {
    gg.AnonymousTraversal
    Traversal func() *Traversal
}

var t_ iAnonymousTraversal = &AnonymousTraversal{
    gg.T__,
    func () *Traversal {
        return &Traversal{
            gg.NewGraphTraversal(nil, gg.NewBytecode(nil), nil),
        }
    },
}

func (t *Traversal) withId(id string) *Traversal {
    return &Traversal{
        t.Where(gg.T__.Values("rsc_id").Is(gg.P.Eq(id))),
    }
}

func (t *Traversal) withTags(tags model.TagSet) *Traversal {
    return &Traversal{
        t.Where(
            gg.T__.In("describes").Values("name").
            Is(gg.P.Within(ToInterfaceSlice(tags.Inner)...)).
            Count().Is(gg.P.Eq(tags.Len())),
        ),
    }
}

func (t *Traversal) withoutTags(tags model.TagSet) *Traversal {
    return &Traversal{
        t.Where(
            gg.T__.In("describes").Values("name").
            Is(gg.P.Within(ToInterfaceSlice(tags.Inner)...)).
            Count().Is(gg.P.Eq(0)),
        ),
    }
}

func (t *Traversal) removeTags(tags model.TagSet) error {
    ce := t.InE("describes").Where(gg.T__.OutV().Values("name").Is(gg.P.Within(ToInterfaceSlice(tags.Inner)...))).Drop().Iterate()
    return <- ce
}

func (t *Traversal) addTags(tags model.TagSet) error {
    ce := t.As("r").V().HasLabel("tag").
    Where(gg.T__.Values("name").Is(gg.P.Within(ToInterfaceSlice(tags.Inner)...))).As("t").
    MergeE(
        map[interface{}]interface{}{
            (gg.T.Label): "describes",
            (gg.Direction.From):  gg.Merge.OutV,
            (gg.Direction.To):    gg.Merge.InV,
        }).
    Option(gg.Merge.OutV, gg.T__.Select("t")).
    Option(gg.Merge.InV, gg.T__.Select("r")).
    Iterate()
    return <- ce
}

func (t *Traversal) toResource() ([]model.Resource, error) {
	rs, err := t.Project("r", "m", "u", "t").
        By(gg.T__.Values("rsc_id")).
        By(gg.T__.Values("mime")).
        By(gg.T__.Values("upload")).
        By(gg.T__.In("describes").Values("name").Fold()).
        GetResultSet()
	var resources []model.Resource
	if err != nil {
		return nil, errors.New("result set failure")
	}
	for r := range rs.Channel() {
		var resource model.Resource
		m, ok := r.Data.(map[interface{}]interface{})
		if !ok {
			return nil, errors.New("Invalid type top map")
		}
		t, ok := m["t"].([]interface{})
		if !ok {
			return nil, errors.New("Invalid type tag slice")
		}
		ts, err := FromInterfaceSlice[string](t)
		if err != nil {
			return nil, errors.New("Invalid tags tagset slice")
		}
		err = resource.Tags.FromSlice(ts)
		if err != nil {
			return nil, errors.New("Invalid tags tagset")
		}
		resource.Id, ok = m["r"].(string)
		if !ok {
			return nil, errors.New("Invalid type rsc id")
		}
		resource.Mimetype, ok = m["m"].(string)
		if !ok {
			return nil, errors.New("Invalid type mime")
		}
		upload, ok := m["u"].(string)
		if !ok {
			return nil, errors.New("Invalid type upload")
		}
		err = nil
		for _, tf := range backend.TimeFormatP {
			resource.CreatedAt, err = time.Parse(tf, upload)
			if err == nil {
				break
			}
		}
		if err != nil {
			return nil, errors.New("Invalid timestamp (parsing)")
		}
		resources = append(resources, resource)
	}
	return resources, nil
}

func (t *Traversal) part(offset, limit int) *Traversal {
    return &Traversal{
        t.Skip(offset).Limit(limit),
    }
}

type Gremlin struct {
    ts *TraversalSource
    fs backend.FileStore
	remote *gg.DriverRemoteConnection
}

func (g *Gremlin) Connect(ctx context.Context) error {
    return nil;
}

func (g *Gremlin) AddResource(ctx context.Context, f io.Reader, tags model.TagSet) (string, error) {
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
    rsc, err := g.ts.resources().withTags(query.Include).withoutTags(query.Exclude).part(query.Offset, query.Limit).toResource()
    return rsc, err
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
    g.remote.Close()
    return nil
}

func New(config Config, fs backend.FileStore) (*Gremlin, error) {
    return &Gremlin{fs: fs}, nil
}
