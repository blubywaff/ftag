package main

import (
	"github.com/google/uuid"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"time"
	// "github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

var templates *template.Template

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
}

func main() {
	// Load Templates
	templates = template.Must(template.ParseGlob("./pages/*.html"))

	statfs := http.FileServer(http.Dir("./dist"))
	webfs := http.FileServer(http.Dir("./pages"))

	http.HandleFunc("/", landingPage)
	http.Handle("/public/", http.StripPrefix("/public/", statfs))
	http.Handle("/site/", http.StripPrefix("/site/", webfs))
	http.HandleFunc("/site/upload", uploadPage)

	log.Fatal(http.ListenAndServe(":8080", nil))
}
