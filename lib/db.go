package lib

import (
	"bytes"
	"context"
	"io"
	"log"
	"net/http"
	"time"

	"errors"
	"os"

	"github.com/google/uuid"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

var NO_RESULT error = errors.New("database no result")

type DatabaseContext struct {
	neo4jdriver neo4j.DriverWithContext
	njctx       context.Context
	sessiondb   map[string]any
}

func SetInSessionDB(ctx DatabaseContext, id string, record any) error {
	ctx.sessiondb[id] = record
	return nil
}

func GetFromSessionDB(ctx DatabaseContext, id string) (any, error) {
	record, ok := ctx.sessiondb[id]
	if !ok {
		return nil, errors.New("does not exist")
	}
	return record, nil
}

func RemoveFromSessionDB(ctx DatabaseContext, id string) error {
	delete(ctx.sessiondb, id)
	return nil
}

func GenUUID() (string, error) {
	rid, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}
	id := rid.String()
	return id, nil
}

// returns id of the resource if created (err == nil)
func AddFile(ctx DatabaseContext, f io.Reader, tags TagSet) (string, error) {
	id, mimetype, fir := writeFileReversible(f)
	if err := fir.OpError(); err != nil {
		err = errorWithContext{err, "Failed to add file due to file write error"}
		log.Println(err)
		return "", err
	}

	neo4jsession := ctx.neo4jdriver.NewSession(ctx.njctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer neo4jsession.Close(ctx.njctx)
	_, err := neo4jsession.ExecuteWrite(ctx.njctx, func(tx neo4j.ManagedTransaction) (any, error) {
		_, err := tx.Run(ctx.njctx, `
        CREATE (a:Resource {id: $fid, createdAt: datetime($date), type: $type})
        FOREACH (tag in $tags |
            MERGE (t:Tag {name: tag})
            CREATE (t)-[:describes]->(a)
        )`, map[string]any{"fid": id, "tags": tags.inner, "type": mimetype, "date": time.Now().UTC().Format(time.RFC3339)})
		// Couldn't get time.Time values to work so format to string and then back
		if err != nil {
			return nil, err
		}
		return nil, nil
	})
	if err != nil {
		fir.Clean()
		return "", errorWithContext{err, "database failed for file uplaod"}
	}
	return id, nil
}

// Returns id, mimetype, canceller
func writeFileReversible(f io.Reader) (string, string, IntermediateResult) {
	hasFailed := false
	doFail := func() { hasFailed = true }

	// only need 512 because that is the max considered by `http.DetectContentType`
	var bts = make([]byte, 512)
	n, err := f.Read(bts)
	if n == 0 {
		doFail()
		return "", "", IntermediateResultFromError(errorWithContext{err, "empty read for mime type"})
	}
	if err != nil && err != io.EOF {
		doFail()
		return "", "", IntermediateResultFromError(errorWithContext{err, "failed to read for mime type"})
	}
	mimetype := http.DetectContentType(bts)

	id, err := GenUUID()
	if err != nil {
		return "", "", IntermediateResultFromError(errorWithContext{err, "could not create uuid"})
	}

	file, err := os.OpenFile("files/"+id, os.O_WRONLY|os.O_CREATE, os.ModePerm)
	if err != nil {
		doFail()
		return "", "", IntermediateResultFromError(errorWithContext{err, "could not create file"})
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
		return "", "", IntermediateResultFromError(errorWithContext{err, "failed on peek copy"})
	}
	_, err = io.Copy(file, f)
	if err != nil {
		return "", "", IntermediateResultFromError(errorWithContext{err, "failed on full copy"})
	}

	return id, mimetype, IntermediateResult{
		cleanup: func() error {
			if err := os.Remove(file.Name()); err != nil {
				log.Println("could not delete on fail: " + id)
				return err
			}
			return nil
		},
		err: nil,
	}
}

func GetFile(ctx DatabaseContext, id string) (Resource, error) {
	neo4jsession := ctx.neo4jdriver.NewSession(ctx.njctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer neo4jsession.Close(ctx.njctx)
	resource, err := neo4jsession.ExecuteRead(ctx.njctx, func(tx neo4j.ManagedTransaction) (any, error) {
		res, err := tx.Run(ctx.njctx, `
        MATCH (r:Resource {id: $fid})<-[:describes]-(t:Tag)
        WITH collect(r) as r, collect(t) as t
        UNWIND r + t as RR
        RETURN distinct RR
        `, map[string]any{"fid": id})
		if err != nil {
			log.Println("database transaction error")
			return nil, err
		}
		recs, err := res.Collect(ctx.njctx)
		if err != nil {
			log.Println("collect error")
			return nil, err
		}
		var resource Resource
		if len(recs) == 0 {
			return nil, errors.New("no resource of id " + id)
		}
		resource.Tags = make([]string, len(recs)-1)
		for _, rec := range recs {
			a, ok := rec.Get("RR")
			if !ok {
				log.Println("RR error")
				return nil, err
			}
			b, ok := a.(neo4j.Node)
			if !ok {
				log.Println("cast error")
				return nil, err
			}
			l := b.Labels[0]
			if l == "Resource" {
				resource = Resource{Id: b.Props["id"].(string), CreatedAt: b.Props["createdAt"].(time.Time), Mimetype: b.Props["type"].(string)}
			}
			if l == "Tag" {
				resource.Tags = append(resource.Tags, b.Props["name"].(string))
			}
		}
		return resource, nil
	})
	if err != nil {
		return Resource{}, err
	}
	return resource.(Resource), err
}

func ChangeTags(ctx DatabaseContext, addtags TagSet, deltags TagSet, id string) error {
	neo4jsession := ctx.neo4jdriver.NewSession(ctx.njctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer neo4jsession.Close(ctx.njctx)
	_, err := neo4jsession.ExecuteWrite(ctx.njctx, func(tx neo4j.ManagedTransaction) (any, error) {
		_, err := tx.Run(ctx.njctx, `
        MATCH (a:Resource {id: $fid})
        CALL {
            WITH a
            UNWIND $addtags as tag
            MERGE (t:Tag {name: tag})
            MERGE (t)-[:describes]->(a)
        }
        CALL {
            WITH a
            UNWIND $deltags as tag
            MATCH (t:Tag {name: tag})-[d:describes]->(a)
            DELETE d
        }
        `, map[string]any{"fid": id, "addtags": addtags.inner, "deltags": deltags.inner})
		if err != nil {
			return nil, err
		}
		return nil, nil
	})
	if err != nil {
		return err
	}
	return nil
}

func TagQuery(ctx DatabaseContext, includes, excludes TagSet, excludeMode string, index int) (Resource, error) {
	neo4jsession := ctx.neo4jdriver.NewSession(ctx.njctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer neo4jsession.Close(ctx.njctx)
	rsrc, err := neo4jsession.ExecuteWrite(ctx.njctx, func(tx neo4j.ManagedTransaction) (any, error) {
		var expart string
		if excludeMode == "or" {
			expart = `AND none(tag in $extag WHERE exists((:Tag {name: tag})-[:describes]->(a)))`
		} else if excludeMode == "and" {
			expart = `AND ((NOT all(tag in $extag WHERE exists((:Tag {name: tag})-[:describes]->(a)))) OR size($extag) = 0)`
		}
		res, err := tx.Run(
			ctx.njctx,
			`MATCH (a:Resource)
WHERE all(tag in $intag WHERE exists((:Tag {name: tag})-[:describes]->(a)))
`+expart+`
RETURN a as RR ORDER BY a.createdAt DESC SKIP $index LIMIT 1`,
			map[string]any{"intag": includes.inner, "extag": excludes.inner, "index": index},
		)
		if err != nil {
			return nil, err
		}
		recs, err := res.Collect(ctx.njctx)
		if err != nil {
			log.Println("collect error")
			return nil, err
		}
		if len(recs) == 0 {
			return nil, NO_RESULT
		}
		rec := recs[0]
		a, ok := rec.Get("RR")
		if !ok {
			log.Println("RR error")
			return nil, err
		}
		b, ok := a.(neo4j.Node)
		if !ok {
			log.Println("cast error")
			return nil, err
		}
		l := b.Labels[0]
		if l == "Resource" {
			return Resource{Id: b.Props["id"].(string), CreatedAt: b.Props["createdAt"].(time.Time), Mimetype: b.Props["type"].(string)}, nil
		} else {
			return Resource{}, errors.New("wrong RR label")
		}
	})
	if err != nil {
		return Resource{}, err
	}
	return rsrc.(Resource), nil
}

// if the error return is nil, the caller must call returned callback to close the database connection
func ConnectDatabases() (DatabaseContext, func(), error) {
	// Create new ctx
	ctx := context.Background()
	// Load database connection
	driver, err := neo4j.NewDriverWithContext("neo4j://localhost:7687", neo4j.NoAuth())
	if err != nil {
		log.Fatal("cannot connect to database")
		return DatabaseContext{}, func() {}, errorWithContext{err, "cannot connect to database"}
	}

	session := driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	ctx = context.WithValue(ctx, "bluby_db_session", session)
	return DatabaseContext{
			driver,
			ctx,
			make(map[string]any),
		}, func() {
			driver.Close(ctx)
		}, nil
}
