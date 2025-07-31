package gremlin

import (
	"errors"
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
			Property("upload", rsc.CreatedAt.UTC().Format(backend.TimeFormat)).
			Property("mime", rsc.Mimetype).
		As("r").
		V().HasLabel("tag").
		Where(g_.Values("name").Is(gp.Within(ToInterfaceSlice(rsc.Tags.Inner)...))).As("t").
		AddE("describes").From(g_.Select("t")).To(g_.Select("r")),
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
	func() *Traversal {
		return &Traversal{
			gg.NewGraphTraversal(nil, gg.NewBytecode(nil), nil),
		}
	},
}

func (t *Traversal) withId(id string) *Traversal {
	return &Traversal{
		t.Where(gg.T__.Values("rsc_id").Is(gp.Eq(id))),
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
	return <-ce
}

func (t *Traversal) addTags(tags model.TagSet) error {
	ce := t.As("r").V().HasLabel("tag").
		Where(gg.T__.Values("name").Is(gg.P.Within(ToInterfaceSlice(tags.Inner)...))).As("t").
		MergeE(
			map[interface{}]interface{}{
				(gg.T.Label):        "describes",
				(gg.Direction.From): gg.Merge.OutV,
				(gg.Direction.To):   gg.Merge.InV,
			}).
		Option(gg.Merge.OutV, gg.T__.Select("t")).
		Option(gg.Merge.InV, gg.T__.Select("r")).
		Iterate()
	return <-ce
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

func (t *Traversal) toTags() (model.TagSet, error) {
	rs, err := t.Values("name").GetResultSet()
	var res model.TagSet
	if err != nil {
		return res, err
	}

	for r := range rs.Channel() {
		s, ok := r.Data.([]interface{})
		if !ok {
			return res, errors.New("Data slice error")
		}
		ss, err := FromInterfaceSlice[string](s)
		if err != nil {
			return res, err
		}
		res.FromSlice(ss)
	}

	return res, nil
}

func (t *Traversal) part(offset, limit int) *Traversal {
	return &Traversal{
		t.Skip(offset).Limit(limit),
	}
}

func (t *Traversal) sort() *Traversal {
	return &Traversal{t.Order().By(t_.Values("upload"), gs.Desc)}
}
