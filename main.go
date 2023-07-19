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
    "flag"

	"github.com/blubywaff/ftag/lib"
)

var templates *template.Template

var dbctx lib.DatabaseContext

var baseUrlFlag = flag.String("urlbase", "", "Specifies the base url for the server. Should include only path, without origin.")
var baseUrl string

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
            "upload.gohtml",
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
			log.Println("error with upload.gohtml")
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

    var tags lib.TagSet
    badtags := tags.FillFromString(req.FormValue("tags"))

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

        res.Header().Add("location", baseUrl+"/site/edit?session="+sessionId)
        res.WriteHeader(303)
        return
    }
    res.Header().Add("location", baseUrl+req.URL.Path)
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
            "edit.gohtml",
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
			log.Println("error with edit.gohtml", err)
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
    var addtags, deltags lib.TagSet
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
            session.FailedAddTags = addtags.FillFromString(buf.String())
        case "deltags":
            buf := new(bytes.Buffer)
            buf.ReadFrom(part)
            if buf.Len() == 0 {
                continue
            }
            session.FailedDelTags = deltags.FillFromString(buf.String())
        default:
            http.Error(res, "invalid field in form", 400)
            return
        }
    }

    if addtags.Len() == 0 && deltags.Len() == 0 && len(session.FailedAddTags) == 0 && len(session.FailedDelTags) == 0 {
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

        res.Header().Add("location", baseUrl+"/site/edit?session="+newSessionId)
        res.WriteHeader(303)
        return
    }

    res.Header().Add("location", baseUrl+"/site/upload")
    res.WriteHeader(303)
    return
}

func viewPage(res http.ResponseWriter, req *http.Request) {
	if req.Method != "GET" {
        res.WriteHeader(405)
        return
	}
    if !strings.Contains(req.URL.String(), "?") {
        err := templates.ExecuteTemplate(
            res,
            "view.gohtml",
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
            log.Println("error with view.gohtml", err)
        }
        return
    }
    var intag, extag lib.TagSet
    var exmode string
    var index int
    intagstr, ok := req.URL.Query()["intags"]
    if !ok {
        http.Error(res, "Missing include tags field", 400)
        return
    }
    intag.FillFromString(intagstr[0])
    extagstr, ok := req.URL.Query()["extags"]
    if !ok {
        http.Error(res, "Missing exclude tags field", 400)
        return
    }
    extag.FillFromString(extagstr[0])
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
        "view.gohtml",
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
            baseUrl + req.URL.Path + "?number="+strconv.Itoa(index-1) + "&intags="+intagstr[0] + "&extags="+extagstr[0] + "&exmode="+exmode,
            baseUrl + req.URL.Path + "?number="+strconv.Itoa(index+1) + "&intags="+intagstr[0] + "&extags="+extagstr[0] + "&exmode="+exmode,
        },
    ) 
    if err != nil {
        res.WriteHeader(500)
        log.Println("error with view.gohtml", err)
    }
}

func debugMiddleWare(prefix string, next http.Handler) http.Handler {
    return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
        log.Println(prefix + " req url: " + req.URL.String())
        next.ServeHTTP(res, req)
    })
}

func main() {
    // Parse flags
    flag.Parse()

    baseUrl = *baseUrlFlag

	// Load Templates
    templates = template.Must(template.New("").Funcs(map[string]any {"hasPrefix": strings.HasPrefix, "getBaseUrl": func() (string) { return baseUrl }}).ParseGlob("./templates/*.gohtml"))

	// Load database connection
    var dbclose func()()
    var err error
    dbctx, dbclose, err = lib.ConnectDatabases()
    if err != nil {
        log.Fatal(err)
    }
    defer dbclose()

    server := http.NewServeMux()

	statfs := http.FileServer(http.Dir("./dist"))
	filefs := http.FileServer(http.Dir("./files"))

	server.HandleFunc("/", landingPage)
	server.Handle("/public/", http.StripPrefix("/public/", statfs))
	server.Handle("/files/", http.StripPrefix("/files/", filefs))
	server.HandleFunc("/site/upload", uploadPage)
	server.HandleFunc("/site/edit", editPage)
	server.HandleFunc("/site/view", viewPage)


	log.Fatal(http.ListenAndServe(":8080", http.StripPrefix(baseUrl, server)))
}
