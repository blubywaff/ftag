package main

import (
	"context"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

var templates *template.Template

var session neo4j.SessionWithContext

var ctx context.Context

func landingPage(res http.ResponseWriter, req *http.Request) {
	res.WriteHeader(200)
	res.Write([]byte("You have reached blubywaff.com at " + time.Now().UTC().Format("2006-01-02 15:04:05") + "."))
}

func uploadPage(res http.ResponseWriter, req *http.Request) {
	if req.Method == "GET" {
		err := templates.ExecuteTemplate(res, "upload.html", nil)
		if err != nil {
			res.WriteHeader(500)
			log.Println("error with upload.html")
		}
		return
	}
	if req.Method != "POST" {
		res.WriteHeader(405)
		return
	}
	err := req.ParseMultipartForm(1 << 24)
	if err != nil {
		res.WriteHeader(500)
		log.Println("error with multipart form upload")
		return
	}
	rid, err := uuid.NewRandom()
	if err != nil {
		res.WriteHeader(500)
		log.Println("could not create uuid")
		return
	}
	id := rid.String()
	f, _, err := req.FormFile("uploadfile")
	if err != nil {
		res.WriteHeader(500)
		log.Println("could not read upload")
		return
	}
	defer f.Close()
	file, err := os.OpenFile("files/"+id, os.O_WRONLY|os.O_CREATE, os.ModePerm)
	if err != nil {
		res.WriteHeader(500)
		log.Println("could not create file")
		return
	}
	do_del := false
	defer func() {
		if !do_del {
			return
		}
		err := os.Remove(file.Name())
		if err != nil {
			log.Println("could not delete on fail")
		}
	}()
	defer file.Close()
	_, err = io.Copy(file, f)
	if err != nil {
		res.WriteHeader(500)
		log.Println("could not copy file")
		do_del = true
		return
	}
    tagstr := req.FormValue("tags")
    tags := strings.Split(tagstr, ",")
    _, err = session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
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
        log.Fatal(err)
        log.Println("Database failed for file upload");
        return;
    }
}

func main() {
    // Initialize context
    ctx = context.Background()

	// Load Templates
	templates = template.Must(template.ParseGlob("./pages/*.html"))

    // Load database connection
    driver, err := neo4j.NewDriverWithContext("neo4j://localhost:7687", neo4j.NoAuth())
    if err != nil {
        log.Fatal("cannot connect to database")
        return;
    }
    defer driver.Close(ctx)

    session = driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
    defer session.Close(ctx)

	statfs := http.FileServer(http.Dir("./dist"))
	webfs := http.FileServer(http.Dir("./pages"))
    filefs := http.FileServer(http.Dir("./files"))

	http.HandleFunc("/", landingPage)
	http.Handle("/public/", http.StripPrefix("/public/", statfs))
	http.Handle("/site/", http.StripPrefix("/site/", webfs))
	http.Handle("/files/", http.StripPrefix("/files/", filefs))
	http.HandleFunc("/site/upload", uploadPage)

	log.Fatal(http.ListenAndServe(":8080", nil))
}
