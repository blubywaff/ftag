package lib

import (
	"bytes"
	"context"
	"io"
	"log"

	"errors"
	"os"

	"github.com/google/uuid"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type DatabaseContext struct {
	neo4jsession neo4j.SessionWithContext
	njctx        context.Context
	sessiondb    map[string]any
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

func GenUUID() (string, error) {
	rid, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}
	id := rid.String()
	return id, nil
}

// returns id of the resource if created (err == nil)
func AddFile(ctx DatabaseContext, f io.Reader, tags []string) (string, error) {
	hasFailed := false
	doFail := func() { hasFailed = true }

	// only need 512 because that is the max considered by `http.DetectContentType`
	var bts = make([]byte, 512)
	n, err := f.Read(bts)
	if n == 0 {
		doFail()
		return "", errorWithContext{err, "empty read for mime type"}
	}
	if err != nil && err != io.EOF {
		doFail()
		return "", errorWithContext{err, "failed to read for mime type"}
	}
	// TODO restore this
	// mimetype := http.DetectContentType(bts)

	id, err := GenUUID()
	if err != nil {
		return "", errorWithContext{err, "could not create uuid"}
	}

	file, err := os.OpenFile("files/"+id, os.O_WRONLY|os.O_CREATE, os.ModePerm)
	if err != nil {
		doFail()
		return "", errorWithContext{err, "could not create file"}
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
		return "", errorWithContext{err, "failed on peek copy"}
	}
	_, err = io.Copy(file, f)
	if err != nil {
		return "", errorWithContext{err, "failed on full copy"}
	}

	_, err = ctx.neo4jsession.ExecuteWrite(ctx.njctx, func(tx neo4j.ManagedTransaction) (any, error) {
		_, err := tx.Run(ctx.njctx, `
        CREATE (a:Resource {id: $fid})
        FOREACH (tag in $tags |
            MERGE (t:Tag {name: tag})
            CREATE (t)-[:describes]->(a)
        )`, map[string]any{"fid": id, "tags": tags})
		if err != nil {
			return nil, err
		}
		return nil, nil
	})
	if err != nil {
		doFail()
		return "", errorWithContext{err, "database failed for file uplaod"}
	}
	return id, nil
}

func GetFile(ctx DatabaseContext, id string) (Resource, error) {
    resource, err := ctx.neo4jsession.ExecuteRead(ctx.njctx, func(tx neo4j.ManagedTransaction) (any, error) {
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
        resource.tags = make([]string, len(recs)-1)
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
                resource = Resource{ id: b.Props["id"].(string) }
            }
            if l == "Tag" {
                resource.tags = append(resource.tags, b.Props["name"].(string))
            }
        }
        log.Printf("%+v", resource)
        return resource, nil
    })
    return resource.(Resource), err
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
			session,
			ctx,
			make(map[string]any),
		}, func() {
			session.Close(ctx)
			driver.Close(ctx)
		}, nil
}
