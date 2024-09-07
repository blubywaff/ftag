package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var NO_RESULT error = errors.New("database no result")

type ctxkeyDriver int
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
	// Generate id, mimetype
	var bts = make([]byte, 1<<25) // 32 MB
	nRead, err := f.Read(bts)
	if nRead == 0 {
		return "", err
	}
	if err != nil && err != io.EOF {
		return "", err
	}
	mimetype := http.DetectContentType(bts)

	id, err := GenUUID()
	if err != nil {
		return "", err
	}

	dbpool := ctx.Value(ctxkeyDriver(0)).(*pgxpool.Pool)
	tx, err := dbpool.Begin(ctx)
	if err != nil {
		return "", err
	}

	// Make sure dead or interrupted transactions are always rolled back
	defer tx.Rollback(ctx)

	// Create resource
	_, err = tx.Exec(ctx, "INSERT INTO Resource (id, mime, upload, data) VALUES ($1::uuid, $2, $3, $4);", id, mimetype, time.Now().UTC(), bts)
	if err != nil {
		return "", errorWithContext{err, "database failed for resource creation"}
	}

	// Add tags
	var batch pgx.Batch
	for i := 0; i < tags.Len(); i++ {
		batch.Queue("INSERT INTO TagOn (resource_id, tag_id) SELECT $1::uuid, ttt.id FROM (SELECT Tag.id FROM Tag WHERE Tag.name IN $2) AS ttt;", id, tags.inner)
	}
	tx.SendBatch(ctx, &batch)
	tx.Commit(ctx)
	return id, nil
}

func GetFile(ctx context.Context, id string) (Resource, error) {
	dbpool := ctx.Value(ctxkeyDriver(0)).(*pgxpool.Pool)
	tx, err := dbpool.Begin(ctx)
	if err != nil {
		return Resource{}, err
	}
	// Transaction should always be ended somehow
	defer tx.Rollback(ctx)

	var batch pgx.Batch
	_ = batch.Queue("SELECT id, mime, upload FROM Resource WHERE id = $1::uuid;", id)
	_ = batch.Queue("SELECT Tag.name FROM TagOn LEFT JOIN Tag ON TagOn.tag_id = Tag.id WHERE resource_id = $1::uuid;", id)
	br := tx.SendBatch(ctx, &batch)

	// exactly one of these will be written to
	rchan := make(chan Resource)
	echan := make(chan error)
	go func() {
		rows, err := br.Query()
		if err != nil {
			echan <- errorWithContext{err, "getfile query issue"}
			return
		}
		if !rows.Next() {
			echan <- rows.Err()
			return
		}
		var rsrc Resource
		err = rows.Scan(&rsrc.Id, &rsrc.Mimetype, &rsrc.CreatedAt)
		if err != nil {
			echan <- errorWithContext{err, "getfile tag scan issue"}
			return
		}
		rows.Close()
		rows, err = br.Query()
		if err != nil {
			echan <- errorWithContext{err, "getfile query issue"}
			return
		}
		var tags []string
		for rows.Next() {
			var tag string
			err = rows.Scan(&tag)
			if err != nil {
				echan <- errorWithContext{err, "getfile scan issue"}
				return
			}
			tags = append(tags, tag)
		}
		if rows.Err() != nil {
			echan <- errorWithContext{err, "getfile rows issue"}
			return
		}
		rsrc.Tags = tags
		rchan <- rsrc
		return
	}()

	var rsrc Resource

	select {
	case err = <-echan:
		log.Print("get file db err: ", err)
		return rsrc, err
	case rsrc = <-rchan:
		// There's nothing to commit since this is read-only, but this ensures resources are cleaned up
		// since we can't read from a completed transaction, hopefully everything clean
		tx.Commit(context.Background())
		return rsrc, nil
	}
}

func ChangeTags(ctx context.Context, addtags TagSet, deltags TagSet, id string) error {
	dbpool := ctx.Value(ctxkeyDriver(0)).(*pgxpool.Pool)
	tx, err := dbpool.Begin(ctx)
	if err != nil {
		return err
	}
	// Transaction should always be ended somehow
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, "DELETE FROM TagOn WHERE resource_id = $1::uuid AND tag_id IN (SELECT Tag.id FROM Tag WHERE Tag.name = any ($2));", id, deltags.inner)
	if err != nil {
		log.Print("ChangeTags Delete issue ", err)
		return err
	}
	_, err = tx.Exec(ctx, "INSERT INTO TagOn (resource_id, tag_id) SELECT $1::uuid, ttt.id FROM (SELECT Tag.id FROM Tag WHERE Tag.name = any ($2)) AS ttt ON CONFLICT DO NOTHING;", id, addtags.inner)
	if err != nil {
		log.Print("ChangeTags insert issue", err)
		return err
	}
	tx.Commit(ctx)
	return nil
}

func TagQuery(ctx context.Context, includes, excludes TagSet, excludeMode string, index int) (Resource, error) {
	dbpool := ctx.Value(ctxkeyDriver(0)).(*pgxpool.Pool)
	tx, err := dbpool.Begin(ctx)
	if err != nil {
		return Resource{}, errorWithContext{err, "TagQuery tx err"}
	}
	// Transaction should always be ended somehow
	defer tx.Rollback(ctx)

	// core query parts
	cte := `
        WITH tt (id, upload, name) AS (
            SELECT Resource.id, upload, Tag.name
            FROM Resource
            JOIN TagOn
                ON TagOn.resource_id = Resource.id
            JOIN Tag
                ON Tag.id = TagOn.tag_id
        )
        SELECT tt.id, tt.upload
        FROM tt
    `
	inc := `
        WHERE tt.name = any ($%d)
        GROUP BY tt.id, tt.upload
        HAVING COUNT(tt.id) = $%d
    `

	exc := `
        EXCEPT
        SELECT tt.id, tt.upload
        FROM tt
        WHERE name = any ($%d)
    `

	// intermediary variable for query
	qq := cte
	var params []any
	if includes.Len() > 0 {
		qq += inc
		params = append(params, includes.inner, includes.Len())
	}
	if excludes.Len() > 0 {
		qq += exc
		params = append(params, excludes.inner)
	}

	wrap := `
        SELECT id
        FROM (
            SELECT DISTINCT id, upload
            FROM ( %s )
            ORDER BY upload DESC, id ASC
            LIMIT 1
            OFFSET $%%d
        );
    `
	params = append(params, index)

	nums := []any{}
	for i := 1; i <= len(params); i++ {
		nums = append(nums, i)
	}
	query := fmt.Sprintf(fmt.Sprintf(wrap, qq), nums...)
	rows, err := tx.Query(ctx, query, params...)
	if err != nil {
		return Resource{}, errorWithContext{err, "TagQuery query issue"}
	}
	defer rows.Close()
	var id string
	if !rows.Next() {
		return Resource{}, NO_RESULT
	}
	err = rows.Scan(&id)
	if err != nil {
		return Resource{}, errorWithContext{err, "TagQuery id scan issue"}
	}
	return GetFile(ctx, id)
}

// if the error return is nil, the caller must call returned callback to close the database connection
func ConnectDatabases(ctx context.Context) (context.Context, func(), error) {
	config := ctx.Value(ctxkeyConfig(0)).(Config).SQL

	// Load database connection
	dbpool, err := pgxpool.New(context.Background(), config.Url)
	if err != nil {
		return ctx, func() {}, errorWithContext{err, "cannot connect to database"}
	}

	ctx = context.WithValue(ctx, ctxkeyDriver(0), dbpool)
	return ctx, func() {
		dbpool.Close()
	}, nil
}
