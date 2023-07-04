package lib

import (
	"bytes"
	"context"
	"io"
	"log"
	// "net/http"
	"os"

	"github.com/google/uuid"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// returns id of the resource
func AddFile(ctx context.Context, f io.Reader, tags []string) (string, error) {
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

	var id string
	{ // uuid
		rid, err := uuid.NewRandom()
		if err != nil {
			return "", errorWithContext{err, "could not create uuid"}
		}
		id = rid.String()
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

	_, err = (ctx.Value("bluby_db_session").(neo4j.SessionWithContext)).ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		_, err := tx.Run(ctx, `
        CREATE (a:File {id: $fid})
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
