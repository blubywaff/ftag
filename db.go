package main

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

type ctxkeyNeo4jDriver int
type ctxkeyMemDB int

type MemDB map[string]any

func SetInSessionDB(ctx context.Context, id string, record any) error {
	ctx.Value(ctxkeyMemDB(0)).(MemDB)[id] = record
	return nil
}

func GetFromSessionDB(ctx context.Context, id string) (any, error) {
	record, ok := ctx.Value(ctxkeyMemDB(0)).(MemDB)[id]
	if !ok {
		return nil, errors.New("does not exist")
	}
	return record, nil
}

func RemoveFromSessionDB(ctx context.Context, id string) error {
	delete(ctx.Value(ctxkeyMemDB(0)).(MemDB), id)
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
func AddFile(ctx context.Context, f io.Reader, tags TagSet) (string, error) {
	id, mimetype, fir := writeFileReversible(f)
	if err := fir.OpError(); err != nil {
		err = errorWithContext{err, "Failed to add file due to file write error"}
		log.Println(err)
		return "", err
	}

	neo4jsession := ctx.Value(ctxkeyNeo4jDriver(0)).(neo4j.DriverWithContext).NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer neo4jsession.Close(ctx)
	_, err := neo4jsession.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		_, err := tx.Run(ctx, `
CREATE (a:Resource {id: $fid, createdAt: datetime($date), type: $type})
FOREACH (tag in $tags |
    MERGE (t:Tag {name: tag})
    CREATE (t)-[:describes]->(a)
)`,
			map[string]any{"fid": id, "tags": tags.inner, "type": mimetype, "date": time.Now().UTC().Format(time.RFC3339)},
		)
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

func GetFile(ctx context.Context, id string) (Resource, error) {
	neo4jsession := ctx.Value(ctxkeyNeo4jDriver(0)).(neo4j.DriverWithContext).NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer neo4jsession.Close(ctx)
	resource, err := neo4jsession.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		res, err := tx.Run(ctx, `
MATCH (r:Resource {id: $fid})<-[:describes]-(t:Tag)
WITH collect(r) as r, collect(t) as t
UNWIND r + t as RR
RETURN distinct RR`,
			map[string]any{"fid": id},
		)
		if err != nil {
			log.Println("database transaction error")
			return nil, err
		}
		recs, err := res.Collect(ctx)
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

func ChangeTags(ctx context.Context, addtags TagSet, deltags TagSet, id string) error {
	neo4jsession := ctx.Value(ctxkeyNeo4jDriver(0)).(neo4j.DriverWithContext).NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer neo4jsession.Close(ctx)
	_, err := neo4jsession.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		_, err := tx.Run(ctx, `
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
}`,
			map[string]any{"fid": id, "addtags": addtags.inner, "deltags": deltags.inner},
		)
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

func TagQuery(ctx context.Context, includes, excludes TagSet, excludeMode string, index int) (Resource, error) {
	neo4jsession := ctx.Value(ctxkeyNeo4jDriver(0)).(neo4j.DriverWithContext).NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer neo4jsession.Close(ctx)
	rsrc, err := neo4jsession.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		var expart string
		if excludes.Len() != 0 {
			if excludeMode == "or" {
				expart = `AND none(tag in $extag WHERE exists((:Tag {name: tag})-[:describes]->(a)))`
			} else if excludeMode == "and" {
				expart = `AND (NOT all(tag in $extag WHERE exists((:Tag {name: tag})-[:describes]->(a))))`
			}
		}
		res, err := tx.Run(
			ctx, `
MATCH (a:Resource)
WHERE all(tag in $intag WHERE exists((:Tag {name: tag})-[:describes]->(a)))
`+expart+`
WITH a ORDER BY a.createdAt DESC, a.id ASC SKIP $index LIMIT 1
CALL {
    WITH a
    MATCH (t:Tag)-[:describes]->(a)
    RETURN collect(a)[0] + collect(t.name) as S0
}
UNWIND S0 as RR
RETURN RR`,
			map[string]any{"intag": includes.inner, "extag": excludes.inner, "index": index},
		)
		if err != nil {
			return nil, err
		}
		recs, err := res.Collect(ctx)
		if err != nil {
			log.Println("collect error")
			return nil, err
		}
		if len(recs) == 0 {
			return nil, NO_RESULT
		}
		var rr Resource
		for _, rec := range recs {
			a, ok := rec.Get("RR")
			if !ok {
				log.Println("RR error")
				return nil, err
			}
			switch b := a.(type) {
			case neo4j.Node:
				rr = Resource{Id: b.Props["id"].(string), CreatedAt: b.Props["createdAt"].(time.Time), Mimetype: b.Props["type"].(string)}
			case string:
				rr.Tags = append(rr.Tags, b)
			default:
				log.Println("cast error")
				log.Printf("%#v", a)
				return nil, errors.New("invalid return type from TagQuery")
			}
		}
		return rr, nil
	})
	if err != nil {
		return Resource{}, err
	}
	return rsrc.(Resource), nil
}

// if the error return is nil, the caller must call returned callback to close the database connection
func ConnectDatabases(ctx context.Context) (context.Context, func(), error) {
	config := ctx.Value(ctxkeyConfig(0)).(Config).Neo4j

	// Load database connection
	driver, err := neo4j.NewDriverWithContext(config.Url, neo4j.BasicAuth(config.Username, config.Password, ""))
	if err != nil {
		return ctx, func() {}, errorWithContext{err, "cannot connect to database"}
	}

	err = driver.VerifyAuthentication(ctx, nil)
	if err != nil {
		driver.Close(ctx)
		return ctx, func() {}, errorWithContext{err, "bad database connection"}
	}

	ctx = context.WithValue(ctx, ctxkeyNeo4jDriver(0), driver)
	return ctx, func() {
		driver.Close(ctx)
	}, nil
}

func CleanDBs(ctx context.Context) error {
	direntries, err := os.ReadDir("files/")
	if err != nil {
		return errorWithContext{err, "Failed to get directory files"}
	}
	neo4jsession := ctx.Value(ctxkeyNeo4jDriver(0)).(neo4j.DriverWithContext).NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer neo4jsession.Close(ctx)
	ids_, err := neo4jsession.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		res, err := tx.Run(ctx, `
MATCH (r:Resource)
RETURN r.id as RR ORDER BY r.id`,
			nil,
		)
		if err != nil {
			log.Println("database transaction error")
			return nil, err
		}
		recs, err := res.Collect(ctx)
		if err != nil {
			log.Println("collect error")
			return nil, err
		}
		var ids []string
		if len(recs) == 0 {
			return ids, nil
		}
		for _, rec := range recs {
			a, ok := rec.Get("RR")
			if !ok {
				log.Println("RR error")
				return nil, err
			}
			b, ok := a.(string)
			if !ok {
				log.Println("cast error")
				return nil, err
			}
			ids = append(ids, b)
		}
		return ids, nil
	})
	if err != nil {
		log.Println("database error", err)
		return err
	}
	dbdel := func(did string) {
		_, err := neo4jsession.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
			_, err := tx.Run(ctx, `
MATCH (r:Resource {id: $fid})
DETACH DELETE r`,
				map[string]any{"fid": did},
			)
			if err != nil {
				log.Println("database transaction error")
				return nil, err
			}
			return nil, nil
		})
		if err != nil {
			log.Println("Could not remove database entry", did, err)
		}
	}
	osdel := func(fid string) {
		log.Println("removing file:", fid)
		err := os.Remove("files/" + fid)
		if err != nil {
			log.Println("Could not remove file", fid, err)
		}
	}
	ids := ids_.([]string)
	var fcur, dcur int
	for fcur < len(direntries) && dcur < len(ids) {
		var fstr, dstr string
		if direntries[fcur].IsDir() {
			fcur++
			continue
		}
		fstr, dstr = direntries[fcur].Name(), ids[dcur]
		if fstr == dstr {
			fcur++
			dcur++
			continue
		}
		if fstr < dstr {
			osdel(fstr)
			fcur++
			continue
		}
		// dstr > fstr
		log.Println("removing dben:", dstr)
		dbdel(dstr)
		dcur++
		continue
	}
	log.Println("Clearing remaining hanging records")
	for fcur == len(direntries) && dcur < len(ids) {
		dbdel(ids[dcur])
		dcur++
	}
	for dcur == len(ids) && fcur < len(direntries) {
		if direntries[fcur].IsDir() {
			fcur++
			continue
		}
		osdel(direntries[fcur].Name())
		fcur++
	}
	return nil
}
