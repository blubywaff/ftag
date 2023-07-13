package main

import (
	"bytes"
	"html/template"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"blubywaff/blubywaff.com/lib"
)

var templates *template.Template

var dbctx lib.DatabaseContext

type PageMeta struct {
    Title string
}

type ClarifySession struct {
    ResourceId string
    FailedAddTags []string
    FailedDelTags []string
}

func landingPage(res http.ResponseWriter, req *http.Request) {
	res.WriteHeader(200)
	res.Write([]byte("You have reached blubywaff.com at " + time.Now().UTC().Format("2006-01-02 15:04:05") + "."))
}

func uploadPage(res http.ResponseWriter, req *http.Request) {
	if req.Method == "GET" {
		err := templates.ExecuteTemplate(
            res,
            "upload.gotmpl",
            struct {
                PageMeta PageMeta
            }{
                PageMeta {
                    "Upload",
                },
            },
        )
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

    tags, badtags := lib.SortTagList(req.FormValue("tags"))

    id, err := lib.AddFile(dbctx, f, tags)
    if err != nil {
        log.Print(err.Error())
        http.Error(res, "Database Error", 500)
        return
    }
    if len(badtags) != 0 {
        sessionId, err := lib.GenUUID()
        if err != nil {
            http.Error(res, "", 500)
            return
        }
        lib.SetInSessionDB(dbctx, sessionId, ClarifySession { ResourceId: id, FailedAddTags: badtags } )

        res.Header().Add("location", "/site/edit?session="+sessionId)
        res.WriteHeader(303)
        return
    }
    res.Header().Add("location", req.URL.Path)
    res.WriteHeader(303)
}

func editPage(res http.ResponseWriter, req *http.Request) {
    if (req.Method != "GET" && req.Method != "POST") {
        res.WriteHeader(405)
        return
    }
    id := req.URL.Query().Get("id")
    sessionId := req.URL.Query().Get("session")
    if id == "" && sessionId == "" {
        http.Error(res, "Must have non-empty id or session", 400)
        return
    }
    var editSession ClarifySession
    if sessionId != "" {
        _editSession, err := lib.GetFromSessionDB(dbctx, sessionId)
        if err != nil {
            http.Error(res, "Invalid session", 400)
            return
        }
        var ok bool
        editSession, ok = _editSession.(ClarifySession)
        if !ok {
            http.Error(res, "Invalid edit session", 400)
            return
        }
        if id != "" && id != editSession.ResourceId {
            http.Error(res, "session and fallback id conflict", 400)
        }
        id = editSession.ResourceId
    }
    if (req.Method == "GET") {
        rsrc, err := lib.GetFile(dbctx, id)
        if err != nil {
            http.Error(res, "Database error", 500)
            return
        }
		err = templates.ExecuteTemplate(
            res,
            "edit.gotmpl",
            struct {
                PageMeta PageMeta
                Resource lib.Resource
                Session ClarifySession
            }{
                PageMeta {
                    "Editing " + id,
                },
                rsrc,
                editSession,
            },
        )
		if err != nil {
			res.WriteHeader(500)
			log.Println("error with edit.gotmpl", err)
            return
		}
		return
    }

    formReader, err := req.MultipartReader()
    if err != nil {
        log.Println("could not open multipart reader", err)
        res.WriteHeader(500)
        return
    }
    var addtags, deltags []string
    var session ClarifySession
    for {
        part, err := formReader.NextPart()
        if err == io.EOF {
            break
        }
        if err != nil {
            log.Println("failed on form part", err)
            res.WriteHeader(500)
            return
        }
        switch part.FormName() {
        case "addtags":
            buf := new(bytes.Buffer)
            buf.ReadFrom(part)
            if buf.Len() == 0 {
                continue
            }
            addtags, session.FailedAddTags = lib.SortTagList(buf.String())
        case "deltags":
            buf := new(bytes.Buffer)
            buf.ReadFrom(part)
            if buf.Len() == 0 {
                continue
            }
            deltags, session.FailedDelTags = lib.SortTagList(buf.String())
        default:
            http.Error(res, "invalid field in form", 400)
            return
        }
    }

    if len(addtags) == 0 && len(deltags) == 0 {
        http.Error(res, "empty form", 400)
        return
    }

    if err := lib.ChangeTags(dbctx, addtags, deltags, id); err != nil {
        log.Println("database failure on changetags", err)
        res.WriteHeader(500)
        return
    }

    if len(session.FailedAddTags) != 0 || len(session.FailedDelTags) != 0 {
        newSessionId, err := lib.GenUUID()
        if err != nil {
            http.Error(res, "", 500)
            return
        }
        lib.RemoveFromSessionDB(dbctx, sessionId)
        lib.SetInSessionDB(dbctx, newSessionId, ClarifySession { ResourceId: id, FailedAddTags: session.FailedAddTags, FailedDelTags: session.FailedDelTags } )

        res.Header().Add("location", "/site/edit?session="+newSessionId)
        res.WriteHeader(303)
        return
    }

    res.Header().Add("location", "/site/upload")
    res.WriteHeader(303)
    return
}

func viewPage(res http.ResponseWriter, req *http.Request) {
	if req.Method != "GET" {
        res.WriteHeader(405)
        return
	}
    // TODO come up with better way to determine form or not
    if !strings.Contains(req.URL.String(), "?") {
        err := templates.ExecuteTemplate(
            res,
            "view.gotmpl",
            struct {
                PageMeta PageMeta;
                Resource interface{};
                Index string;
            } {
                PageMeta {
                    Title: "Viewer",
                },
                nil,
                "",
            },
        ) 
        if err != nil {
            res.WriteHeader(500)
            log.Println("error with view.gotmpl", err)
        }
        return
    }
    var intag, extag []string
    var exmode string
    var index int
    intagstr, ok := req.URL.Query()["intags"]
    if !ok {
        http.Error(res, "Missing include tags field", 400)
        return
    }
    intag, _ = lib.SortTagList(intagstr[0])
    extagstr, ok := req.URL.Query()["extags"]
    if !ok {
        http.Error(res, "Missing exclude tags field", 400)
        return
    }
    extag, _ = lib.SortTagList(extagstr[0])
    exmodestr, ok := req.URL.Query()["exmode"]
    if !ok {
        http.Error(res, "Missing exmode field", 400)
        return
    }
    exmode = exmodestr[0]
    if exmode != "or" && exmode != "and" {
        http.Error(res, "Invalid exlude mode", 400)
        return
    }
    numerstr, ok := req.URL.Query()["number"]
    if !ok {
        http.Error(res, "Missing number field", 400)
        return
    }
    index, err := strconv.Atoi(numerstr[0])
    if err != nil {
        http.Error(res, "invalid number", 400)
        return
    }
    if index < 1 {
        http.Error(res, "exceed list beginning", 400)
        return
    }
    rsrc, err := lib.TagQuery(dbctx, intag, extag, exmode, index - 1)
    if err == lib.NO_RESULT {
        if index == 1 {
            http.Error(res, "no result", 400)
            return
        }
        http.Error(res, "exceed list end", 400)
        return
    }
    if err != nil {
        res.WriteHeader(500)
        log.Println("err with viewPage db TagQuery", err)
        return
    }
    err = templates.ExecuteTemplate(
        res,
        "view.gotmpl",
        struct {
            PageMeta PageMeta;
            Resource lib.Resource;
            PrevLink string;
            NextLink string;
        } {
            PageMeta {
                Title: "Viewing " + rsrc.Id,
            },
            rsrc,
            req.URL.Path + "?number="+strconv.Itoa(index-1) + "&intags="+intagstr[0] + "&extags="+extagstr[0] + "&exmode="+exmode,
            req.URL.Path + "?number="+strconv.Itoa(index+1) + "&intags="+intagstr[0] + "&extags="+extagstr[0] + "&exmode="+exmode,
        },
    ) 
    if err != nil {
        res.WriteHeader(500)
        log.Println("error with view.gotmpl", err)
    }
}

func main() {
	// Load Templates
    templates = template.Must(template.New("").Funcs(map[string]any {"hasPrefix": strings.HasPrefix}).ParseGlob("./templates/*.gotmpl"))

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
	http.HandleFunc("/site/edit", editPage)
	http.HandleFunc("/site/view", viewPage)

	log.Fatal(http.ListenAndServe(":8080", nil))
}
