package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

var templates *template.Template

var (
	INVALID_FORM_FIELD       = errors.New("invalid field in form")
	EMPTY_FORM               = errors.New("empty form")
	MISSING_FORM_REQUIREMENT = errors.New("required form field not present")
)

type ctxkeyConfig int
type ctxkeyUserSettings int

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
				PageMeta{
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
	config := req.Context().Value(ctxkeyConfig(0)).(Config)
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

	var tags TagSet
	badtags := tags.FillFromString(req.FormValue("tags"))

	id, err := AddFile(req.Context(), f, tags)
	if err != nil {
		log.Print(err.Error())
		http.Error(res, "Database Error", 500)
		return
	}
	if len(badtags) != 0 {
		sessionId, err := GenUUID()
		if err != nil {
			http.Error(res, "", 500)
			return
		}
		SetInSessionDB(req.Context(), sessionId, ClarifySession{ResourceId: id, FailedAddTags: badtags})

		res.Header().Add("location", config.UrlBase+"/site/edit?session="+sessionId)
		res.WriteHeader(303)
		return
	}
	res.Header().Add("location", config.UrlBase+"/site/edit?id="+id)
	res.WriteHeader(303)
}

func multiuploadPage(res http.ResponseWriter, req *http.Request) {
	if req.Method == "GET" {
		err := templates.ExecuteTemplate(
			res,
			"multiupload.gohtml",
			struct {
				PageMeta PageMeta
			}{
				PageMeta{
					"Upload",
				},
			},
		)
		if err != nil {
			res.WriteHeader(500)
			log.Println("error with multiupload.gohtml")
		}
		return
	}
	if req.Method != "POST" {
		res.WriteHeader(405)
		return
	}
	config := req.Context().Value(ctxkeyConfig(0)).(Config)
	// 1024 megabytes
	// consider using maltipart reader to avoid reading oversized uploads
	err := req.ParseMultipartForm(1 << 30)
	if err != nil {
		res.WriteHeader(500)
		log.Println("error with multipart form upload")
		return
	}

	var tags TagSet
	badtags := tags.FillFromString(req.FormValue("tags"))
	if len(badtags) != 0 {
		http.Error(res, "Some tags were invalid, multiupload aborted.", 400)
		return
	}

	fhs := req.MultipartForm.File["uploadfile"]
	for _, fh := range fhs {
		f, err := fh.Open()
		if err != nil {
			log.Println("failed to open file from fileheader", err)
			continue // safety measure TODO figure this out
		}
		defer f.Close()
		_, err = AddFile(req.Context(), f, tags)
		if err != nil {
			log.Println("failed to write file to database", err)
			continue // TODO there should be some failure mode here
		}
	}

	res.Header().Add("location", config.UrlBase+"/site/view")
	res.WriteHeader(303)
}

func editreqLogic(req *http.Request) (ClarifySession, error) {
	formReader, err := req.MultipartReader()
	if err != nil {
		err = errorWithContext{err, "could not open multipart reader"}
		log.Println(err)
		return ClarifySession{}, err
	}
	var addtags, deltags TagSet
	var session ClarifySession
	for {
		part, err := formReader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			err = errorWithContext{err, "failed on form part"}
			return ClarifySession{}, err
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
		case "resourceid":
			buf := new(bytes.Buffer)
			buf.ReadFrom(part)
			if buf.Len() == 0 {
				continue
			}
			session.ResourceId = buf.String()
		default:
			return ClarifySession{}, INVALID_FORM_FIELD
		}
	}

	if session.ResourceId == "" {
		return ClarifySession{}, MISSING_FORM_REQUIREMENT
	}

	if addtags.Len() == 0 && deltags.Len() == 0 && len(session.FailedAddTags) == 0 && len(session.FailedDelTags) == 0 {
		return ClarifySession{}, EMPTY_FORM
	}

	if err := ChangeTags(req.Context(), addtags, deltags, session.ResourceId); err != nil {
		err = errorWithContext{err, "database failure on changetags"}
		return ClarifySession{}, err
	}
	return session, nil
}

func editPage(res http.ResponseWriter, req *http.Request) {
	if req.Method != "GET" && req.Method != "POST" {
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
		_editSession, err := GetFromSessionDB(req.Context(), sessionId)
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
	if req.Method == "GET" {
		rsrc, err := GetFile(req.Context(), id)
		if err != nil {
			http.Error(res, "Database error", 500)
			return
		}
		err = templates.ExecuteTemplate(
			res,
			"edit.gohtml",
			struct {
				PageMeta PageMeta
				Resource Resource
				Session  ClarifySession
			}{
				PageMeta{
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

	config := req.Context().Value(ctxkeyConfig(0)).(Config)

	session, err := editreqLogic(req)
	if err == nil {
		goto editpage_logic_noerr
	}
	if err == EMPTY_FORM {
		http.Error(res, "empty form", 400)
		return
	}
	if err == INVALID_FORM_FIELD {
		http.Error(res, "invalid field in form", 400)
		return
	}

editpage_logic_noerr:
	if len(session.FailedAddTags) != 0 || len(session.FailedDelTags) != 0 {
		newSessionId, err := GenUUID()
		if err != nil {
			http.Error(res, "", 500)
			return
		}
		RemoveFromSessionDB(req.Context(), sessionId)
		SetInSessionDB(req.Context(), newSessionId, ClarifySession{ResourceId: id, FailedAddTags: session.FailedAddTags, FailedDelTags: session.FailedDelTags})

		res.Header().Add("location", config.UrlBase+"/site/edit?session="+newSessionId)
		res.WriteHeader(303)
		return
	}

	res.Header().Add("location", config.UrlBase+"/site/edit?id="+id)
	res.WriteHeader(303)
	return
}

func viewPage(res http.ResponseWriter, req *http.Request) {
	config := req.Context().Value(ctxkeyConfig(0)).(Config)
	if req.Method == "POST" {
		_, err := editreqLogic(req)
		if err != nil {
			err = errorWithContext{err, "failure of editreqlogic for view page"}
			res.WriteHeader(500)
			return
		}
		res.Header().Add("location", config.UrlBase+req.URL.RequestURI())
		res.WriteHeader(303)
		return
	}
	if req.Method != "GET" {
		res.WriteHeader(405)
		return
	}
	if !strings.Contains(req.URL.String(), "?") {
		err := templates.ExecuteTemplate(
			res,
			"view.gohtml",
			struct {
				PageMeta PageMeta
				Resource interface{}
				Index    string
			}{
				PageMeta{
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
	var intag, extag TagSet
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
	ust := req.Context().Value(ctxkeyUserSettings(0)).(UserSettings)
	// Adds all user default exclusions that are not specifically included
	extag.Union(*ust.View.DefaultExcludes.Duplicate().Difference(intag))
	rsrc, err := TagQuery(req.Context(), intag, extag, exmode, index-1)
	if err == NO_RESULT {
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
			PageMeta     PageMeta
			Resource     Resource
			PrevLink     string
			NextLink     string
			UserSettings UserSettings
		}{
			PageMeta{
				Title: "Viewing " + rsrc.Id,
			},
			rsrc,
			config.UrlBase + req.URL.Path + "?number=" + strconv.Itoa(index-1) + "&intags=" + intagstr[0] + "&extags=" + extagstr[0] + "&exmode=" + exmode,
			config.UrlBase + req.URL.Path + "?number=" + strconv.Itoa(index+1) + "&intags=" + intagstr[0] + "&extags=" + extagstr[0] + "&exmode=" + exmode,
			ust,
		},
	)
	if err != nil {
		res.WriteHeader(500)
		log.Println("error with view.gohtml", err)
	}
}

func settingsPage(res http.ResponseWriter, req *http.Request) {
	if req.Method == "GET" {
		us := req.Context().Value(ctxkeyUserSettings(0)).(UserSettings)
		err := templates.ExecuteTemplate(res, "settings.gohtml", struct {
			PageMeta     PageMeta
			UserSettings UserSettings
		}{
			PageMeta{"Settings"},
			us,
		})
		if err != nil {
			log.Println("error with settings.gothml:", err)
			return
		}
		return
	}
	if req.Method != "POST" {
		res.WriteHeader(405)
		return
	}
	var ust UserSettings
	req.ParseForm()
	ust.View.TagVisibility = req.FormValue("view-tags")
	ust.View.DefaultExcludes.FillFromString(req.FormValue("def-ex"))
	if err := ust.Verify(); err != nil {
		log.Println("Failed to verify settings submission", err)
		http.Error(res, "Invalid value", 400)
		return
	}
	str, err := ust.ToCookieString()
	if err != nil {
		log.Println("Failed to get cookie string", err)
		res.WriteHeader(500)
		return
	}
	http.SetCookie(res, &http.Cookie{Name: "settings", Value: str})
	res.Header().Add("location", req.Context().Value(ctxkeyConfig(0)).(Config).UrlBase+"/site/settings")
	res.WriteHeader(303)
}

func debugMiddleWare(prefix string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		log.Println(prefix + " req url: " + req.URL.String())
		next.ServeHTTP(res, req)
	})
}

func addContext(ctx context.Context, next http.Handler) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		next.ServeHTTP(res, req.WithContext(ctx))
	})
}

func attachUserSettings(next http.Handler) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		var userSettings UserSettings
		cookie, err := req.Cookie("settings")
		if err == http.ErrNoCookie {
			userSettings = DefaultUserSettings
			goto gonext
		}
		err = userSettings.FromCookieString(cookie.Value)
		if err != nil {
			log.Println("Could not get userSettings:", err)
		}
	gonext:
		next.ServeHTTP(res, req.WithContext(context.WithValue(ctx, ctxkeyUserSettings(0), userSettings)))
	})
}

func main() {
	// Declare limited flags
	var (
		cleanupFlag    = flag.Bool("clean", false, "If the database should be cleaned on startup.")
		configPathFlag = flag.String("config", "ftag.config.json", "The location of the config file.")
	)

	// Parse flags
	flag.Parse()

	// Setup Context
	var ctx = context.Background()

	// Parse Config
	bts, err := os.ReadFile(*configPathFlag)
	if err != nil {
		log.Fatal("failed to read config:", err)
	}
	var config Config
	json.Unmarshal(bts, &config)
	ctx = context.WithValue(ctx, ctxkeyConfig(0), config)

	// Load Templates
	templates = template.Must(template.New("").Funcs(map[string]any{
		"hasPrefix":   strings.HasPrefix,
		"getBaseUrl":  func() string { return config.UrlBase },
		"stringifyTS": func(ts TagSet) string { return ts.String() },
	}).ParseGlob("./templates/*.gohtml"))

	// Load database connection
	var dbclose func()
	ctx, dbclose, err = ConnectDatabases(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer dbclose()

	if *cleanupFlag {
		err := CleanDBs(ctx)
		if err != nil {
			log.Println("Failed to clean up dbs", err)
		} else {
			log.Println("Cleaned databases")
		}
	}

	server := http.NewServeMux()

	statfs := http.FileServer(http.Dir("./dist"))
	filefs := http.FileServer(http.Dir("./files"))

	server.HandleFunc("/", landingPage)
	server.Handle("/public/", http.StripPrefix("/public/", statfs))
	server.Handle("/files/", http.StripPrefix("/files/", filefs))
	server.HandleFunc("/site/upload", uploadPage)
	server.HandleFunc("/site/upload/many", multiuploadPage)
	server.HandleFunc("/site/edit", editPage)
	server.HandleFunc("/site/view", viewPage)
	server.HandleFunc("/site/settings", settingsPage)

	log.Fatal(http.ListenAndServe(":8080", addContext(ctx, attachUserSettings(http.StripPrefix(config.UrlBase, server)))))
}
