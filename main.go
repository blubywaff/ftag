package main

import (
	"context"
	"html/template"
	"log"
	"net/http"
	"strings"
	"time"

	"blubywaff/blubywaff.com/lib"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

var templates *template.Template

var ctx context.Context

func landingPage(res http.ResponseWriter, req *http.Request) {
	res.WriteHeader(200)
	res.Write([]byte("You have reached blubywaff.com at " + time.Now().UTC().Format("2006-01-02 15:04:05") + "."))
}

func uploadPage(res http.ResponseWriter, req *http.Request) {
	writePage := func() {
		err := templates.ExecuteTemplate(res, "upload.gotmpl", nil)
		if err != nil {
			res.WriteHeader(500)
			log.Println("error with upload.gotmpl")
		}
	}
	if req.Method == "GET" {
		writePage()
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
	f, _, err := req.FormFile("uploadfile")
	if err != nil {
		res.WriteHeader(500)
		log.Println("could not read upload")
		return
	}
	defer f.Close()
	tagstr := req.FormValue("tags")
	tags := strings.Split(tagstr, ",")
	// TODO vet tags
	lib.AddFile(ctx, f, tags)

	writePage()
}

func viewPage(res http.ResponseWriter, req *http.Request) {
	if req.Method == "GET" {
		err := templates.ExecuteTemplate(res, "view.gotmpl", nil)
		if err != nil {
			res.WriteHeader(500)
			log.Println("error with view.gotmpl")
		}
		return
	}
}

func main() {
	// Initialize context
	ctx = context.Background()

	// Load Templates
	templates = template.Must(template.ParseGlob("./templates/*.gotmpl"))

	// Load database connection
	driver, err := neo4j.NewDriverWithContext("neo4j://localhost:7687", neo4j.NoAuth())
	if err != nil {
		log.Fatal("cannot connect to database")
		return
	}
	defer driver.Close(ctx)

	session := driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	ctx = context.WithValue(ctx, "bluby_db_session", session)
	defer session.Close(ctx)

	statfs := http.FileServer(http.Dir("./dist"))
	filefs := http.FileServer(http.Dir("./files"))

	http.HandleFunc("/", landingPage)
	http.Handle("/public/", http.StripPrefix("/public/", statfs))
	http.Handle("/files/", http.StripPrefix("/files/", filefs))
	http.HandleFunc("/site/upload", uploadPage)
	http.HandleFunc("/site/view", viewPage)

	log.Fatal(http.ListenAndServe(":8080", nil))
}
