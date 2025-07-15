package db

import (
	"bytes"
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"errors"

	gremlingo "github.com/apache/tinkerpop/gremlin-go/v3/driver"
	"github.com/blubywaff/ftag/internal/config"
	"github.com/blubywaff/ftag/internal/error"
	"github.com/blubywaff/ftag/internal/model"
	"github.com/google/uuid"
)

var TimeFormat = time.RFC3339
var TimeFormatP = []string{time.RFC3339, time.RFC3339Nano, "2006-01-02T15:04:05"}

// common
var __ = gremlingo.T__
var TextP = gremlingo.TextP

type GraphTraversal = gremlingo.GraphTraversal
type GraphTraversalSource = gremlingo.GraphTraversalSource
type DriverRemoteConnection = gremlingo.DriverRemoteConnection

// predicates
var between = gremlingo.P.Between
var eq = gremlingo.P.Eq
var gt = gremlingo.P.Gt
var gte = gremlingo.P.Gte
var inside = gremlingo.P.Inside
var lt = gremlingo.P.Lt
var lte = gremlingo.P.Lte
var neq = gremlingo.P.Neq
var not = gremlingo.P.Not
var outside = gremlingo.P.Outside
var test = gremlingo.P.Test
var within = gremlingo.P.Within
var without = gremlingo.P.Without
var and = gremlingo.P.And
var or = gremlingo.P.Or

// sorting
var order = gremlingo.Order

// selects
var keys = gremlingo.Column.Keys
var values = gremlingo.Column.Values
var outV = gremlingo.Merge.OutV
var inV = gremlingo.Merge.InV
var label = gremlingo.T.Label
var from = gremlingo.Direction.From
var to = gremlingo.Direction.To
var desc = gremlingo.Order.Desc
var asc = gremlingo.Order.Asc
var NO_RESULT error = errors.New("database no result")

type Database interface {
	// returns the id of the newly added file
	AddFile(ctx context.Context, f io.Reader, tags model.TagSet) (string, error)
	ChangeTags(ctx context.Context, addtags model.TagSet, deltags model.TagSet, id string) error
	TagQuery(ctx context.Context, query model.Query) ([]model.Resource, error)
	GetFile(ctx context.Context, id string) (model.Resource, error)
	GetBytes(ctx context.Context, id string) ([]byte, error)
	Close(ctx context.Context) error
}

func GenUUID() (string, error) {
	rid, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}
	id := rid.String()
	return id, nil
}

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

type Tinkerpop struct {
	g      *GraphTraversalSource
	remote *gremlingo.DriverRemoteConnection
}

func ToResources(g *GraphTraversal) ([]model.Resource, error) {
	rs, err := g.GetResultSet()
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
		v, ok := m["r"].(map[interface{}]interface{})
		if !ok {
			return nil, errors.New("Invalid type resource map")
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
		resource.Id, ok = v["rsc_id"].(string)
		if !ok {
			return nil, errors.New("Invalid type rsc id")
		}
		resource.Mimetype, ok = v["mime"].(string)
		if !ok {
			return nil, errors.New("Invalid type mime")
		}
		upload, ok := v["upload"].(string)
		if !ok {
			return nil, errors.New("Invalid type upload")
		}
		err = nil
		for _, tf := range TimeFormatP {
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

func (t *Tinkerpop) AddFile(ctx context.Context, f io.Reader, tags model.TagSet) (string, error) {
	tx := t.g.Tx()
	g, err := tx.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	uid, mime, ir := writeFileReversible(f)
	if err := ir.OpError(); err != nil {
		return "", err
	}
	defer ir.Clean()
	resource_map := make(map[string]string)
	resource_map["r"] = uid
	resource_map["m"] = mime
	resource_map["u"] = time.Now().UTC().Format(TimeFormat)
	ce := g.Inject(resource_map).
		AddV("resource").
		Property("rsc_id", __.Select("r")).
		Property("mime", __.Select("m")).
		Property("upload", __.Select("u")).As("r").
		V().HasLabel("tag").
		Where(__.Values("name").Is(within(ToInterfaceSlice(tags.Inner)...))).As("t").
		AddE("describes").From(__.Select("t")).To(__.Select("r")).
		Iterate()
	err = <-ce
	if err != nil {
		return "", err
	}
	err = tx.Commit()
	if err != nil {
		return "", err
	}
	err = ir.Commit()
	if err != nil {
		return "", err
	}
	return "", nil
}

// Returns id, mimetype, canceller
func writeFileReversible(f io.Reader) (string, string, apperror.IntermediateResult) {
	hasFailed := false
	doFail := func() { hasFailed = true }

	// only need 512 because that is the max considered by `http.DetectContentType`
	var bts = make([]byte, 512)
	n, err := f.Read(bts)
	if n == 0 {
		doFail()
		return "", "", apperror.IntermediateResultFromError(apperror.ErrorWithContext{Original: err, Message: "empty read for mime type"})
	}
	if err != nil && err != io.EOF {
		doFail()
		return "", "", apperror.IntermediateResultFromError(apperror.ErrorWithContext{Original: err, Message: "failed to read for mime type"})
	}
	mimetype := http.DetectContentType(bts)

	id, err := GenUUID()
	if err != nil {
		return "", "", apperror.IntermediateResultFromError(apperror.ErrorWithContext{Original: err, Message: "could not create uuid"})
	}

	file, err := os.OpenFile("files/"+id, os.O_WRONLY|os.O_CREATE, os.ModePerm)
	if err != nil {
		doFail()
		return "", "", apperror.IntermediateResultFromError(apperror.ErrorWithContext{Original: err, Message: "could not create file"})
	}
	defer func(_id string) {
		if err := file.Close(); err != nil {
			// This represents a serious program issue
			log.Panicln("UNEX double close")
		}
		if !hasFailed {
			return
		}
		if err := os.Remove(file.Name()); err != nil {
			log.Println("could not delete on fail: " + _id)
		}
	}(id)

	_, err = io.Copy(file, bytes.NewReader(bts))
	if err != nil {
		return "", "", apperror.IntermediateResultFromError(apperror.ErrorWithContext{Original: err, Message: "failed on peek copy"})
	}
	_, err = io.Copy(file, f)
	if err != nil {
		return "", "", apperror.IntermediateResultFromError(apperror.ErrorWithContext{Original: err, Message: "failed on full copy"})
	}

	return id, mimetype, apperror.IntermediateResult{
		Cleanup: func() error {
			if err := os.Remove(file.Name()); err != nil {
				log.Println("could not delete on fail: " + id)
				return err
			}
			return nil
		},
		Err: nil,
	}
}

func (t *Tinkerpop) TagQuery(ctx context.Context, query model.Query) ([]model.Resource, error) {
	var gt *GraphTraversal
	if query.Include.Len() == 0 {
		gt = t.g.V().HasLabel("resource")
	} else {
		gt = t.g.V().HasLabel("tag").
			Where(__.Values("name").Is(within(ToInterfaceSlice(query.Include.Inner)...))).
			Out("describes").GroupCount().Unfold().
			Where(__.Select(values).Is(eq(query.Include.Len()))).
			Select(keys)
	}

	val := gt.As("r").In("describes").Values("name").
		Group().By(__.Select("r")).Unfold().
		Where(
			__.Select(values).
				All(not(within(ToInterfaceSlice(query.Exclude.Inner)...)))).
		Order().By(__.Select(keys).Values("upload"), desc).
		Skip(query.Offset).Limit(query.Limit).
		As("r").Project("r", "t").
		By(__.Select(keys).ElementMap()).
		By(__.Select(values))

	return ToResources(val)
}

func (t *Tinkerpop) GetFile(ctx context.Context, id string) (model.Resource, error) {
	tr := t.g.V().
		Has("resource", "rsc_id", id).
		As("r").
		In("describes").
		Values("name").
		Group().
		By(__.Select("r")).
		Unfold().
		Project("r", "t").
		By(__.Select(keys).ElementMap()).
		By(__.Select(values))

	resources, err := ToResources(tr)
	if err != nil {
		return model.Resource{}, err
	}
	return resources[0], nil
}

func (t *Tinkerpop) ChangeTags(ctx context.Context, addtags model.TagSet, deltags model.TagSet, id string) error {
	tx := t.g.Tx()
	g, err := tx.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	ce := g.V().HasLabel("resource").
		Where(__.Values("rsc_id").Is(within([]interface{}{id}...))).As("r").
		V().HasLabel("tag").
		Where(__.Values("name").Is(within(ToInterfaceSlice(addtags.Inner)...))).As("t").
		MergeE(
			map[interface{}]interface{}{
				(label): "describes",
				(from):  outV,
				(to):    inV,
			}).
		Option(outV, __.Select("t")).
		Option(inV, __.Select("r")).
		Iterate()
	err = <-ce
	if err != nil {
		return err
	}
	ce = g.V().HasLabel("resource").
		Where(__.Values("rsc_id").Is(within([]interface{}{id}...))).
		InE("describes").
		Where(__.OutV().Values("name").Is(within(ToInterfaceSlice(deltags.Inner)...))).
		Drop().
		Iterate()
	err = <-ce
	if err != nil {
		return err
	}
	tx.Commit()
	return nil
}

func (t *Tinkerpop) GetBytes(ctx context.Context, id string) ([]byte, error) {
	return os.ReadFile("./files/" + id)
}

func (t *Tinkerpop) Close(ctx context.Context) error {
	t.remote.Close()
	return nil
}

func ConnectDatabases(ctx context.Context) (*Tinkerpop, error) {
	config := config.Global.Gremlin

	var result Tinkerpop

	remote, err := gremlingo.NewDriverRemoteConnection(config.Url)
	result.g = gremlingo.Traversal_().WithRemote(remote)
	result.remote = remote

	return &result, err
}
