package main

import (
	"html/template"
	"log"
	"net/http"
	"strings"
	"time"

	"blubywaff/blubywaff.com/lib"
)

var templates *template.Template

var dbctx lib.DatabaseContext

func landingPage(res http.ResponseWriter, req *http.Request) {
	res.WriteHeader(200)
	res.Write([]byte("You have reached blubywaff.com at " + time.Now().UTC().Format("2006-01-02 15:04:05") + "."))
}

func uploadPage(res http.ResponseWriter, req *http.Request) {
	if req.Method == "GET" {
		err := templates.ExecuteTemplate(res, "upload.gotmpl", nil)
		if err != nil {
			res.WriteHeader(500)
			log.Println("error with upload.gotmpl")
		}
		return
	}
	if req.Method != "POST" {
		res.WriteHeader(405)
		return
	}
    // 64 megabytes
    // consider using maltipart reader to avoid reading oversized uploads
	err := req.ParseMultipartForm(1 << 26)
	if err != nil {
		res.WriteHeader(500)
		log.Println("error with multipart form upload")
		return
	}
	f, _, err := req.FormFile("uploadfile")
	if err != nil {
		res.WriteHeader(500)
		log.Println("could not read upload " + err.Error())
		return
	}
	defer f.Close()

    var tags []string
    {
        tagstr := req.FormValue("tags")
        _tags := strings.Split(tagstr, ",")
        tags = make([]string, 0, len(_tags))
        for _, t := range _tags {
            if c, _ := lib.DoesTagConform(t); c {
                tags = append(tags, t)
                continue
            }
            // tag with error TODO
        }
    }

    lib.AddFile(dbctx, f, tags)
    // TODO add upload session for edit page

    res.Header().Add("location", req.URL.Path)
    res.WriteHeader(303)
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
	// Load Templates
	templates = template.Must(template.ParseGlob("./templates/*.gotmpl"))

	// Load database connection
    var dbclose func()()
    var err error
    dbctx, dbclose, err = lib.ConnectDatabases()
    if err != nil {
        log.Fatal(err)
    }
    defer dbclose()

	statfs := http.FileServer(http.Dir("./dist"))
	filefs := http.FileServer(http.Dir("./files"))

	http.HandleFunc("/", landingPage)
	http.Handle("/public/", http.StripPrefix("/public/", statfs))
	http.Handle("/files/", http.StripPrefix("/files/", filefs))
	http.HandleFunc("/site/upload", uploadPage)
	http.HandleFunc("/site/view", viewPage)

	log.Fatal(http.ListenAndServe(":8080", nil))
}
