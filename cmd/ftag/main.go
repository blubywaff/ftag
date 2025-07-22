package main

import (
	"context"
	"encoding/json"
	"errors"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/blubywaff/ftag/internal/config"
	"github.com/blubywaff/ftag/internal/db"
	"github.com/blubywaff/ftag/internal/model"
)

var templates *template.Template

var client db.Database

var (
	INVALID_FORM_FIELD       = errors.New("invalid field in form")
	EMPTY_FORM               = errors.New("empty form")
	MISSING_FORM_REQUIREMENT = errors.New("required form field not present")
)

type TagChange struct {
	AddTags    string
	DelTags    string
	ResourceId string
}

func writeJson[T any](res http.ResponseWriter, value T) {
	bts, err := json.Marshal(value)
	if err != nil {
		res.WriteHeader(500)
		log.Println("error with marshaling", err)
		return
	}
	_, err = res.Write(bts)
	if err != nil {
		res.WriteHeader(500)
		log.Println("error with view.gohtml", err)
	}
}

func landingPage(res http.ResponseWriter, req *http.Request) {
	res.WriteHeader(200)
	res.Write([]byte("You have reached blubywaff.com at " + time.Now().UTC().Format("2006-01-02 15:04:05") + "."))
}

func multiuploadPage(res http.ResponseWriter, req *http.Request) {
	if req.Method == "GET" {
		err := templates.ExecuteTemplate(
			res,
			"multiupload.gohtml",
			struct {
				PageMeta model.PageMeta
			}{
				model.PageMeta{
					Title: "Upload",
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
	// 1024 megabytes
	// consider using maltipart reader to avoid reading oversized uploads
	err := req.ParseMultipartForm(1 << 30)
	if err != nil {
		res.WriteHeader(500)
		log.Println("error with multipart form upload")
		return
	}

	var tags model.TagSet
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
		_, err = client.AddFile(req.Context(), f, tags)
		if err != nil {
			log.Println("failed to write file to database", err)
			continue // TODO there should be some failure mode here
		}
	}

	res.Header().Add("location", config.Global.UrlBase+"/site/view")
	res.WriteHeader(303)
}

func query(res http.ResponseWriter, req *http.Request) {
	if req.Method != "GET" {
		res.WriteHeader(405)
		return
	}
	if !strings.Contains(req.URL.String(), "?") {
		res.WriteHeader(400)
		return
	}
	ust := req.Context().Value(model.CtxkeyUserSettings(0)).(model.UserSettings)
	var intag, extag model.TagSet
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
	// Adds all user default exclusions that are not specifically included
	extag.Union(*ust.View.DefaultExcludes.Duplicate().Difference(intag))
	query := model.Query{Include: intag, Exclude: extag, Offset: index - 1, Limit: 1}
	rsrcs, err := client.TagQuery(req.Context(), query)
	if err != nil {
		res.WriteHeader(500)
		log.Println("err with viewPage db TagQuery", err)
		return
	}
	if len(rsrcs) == 0 {
		if index == 1 {
			http.Error(res, "no result", 400)
			return
		}
		http.Error(res, "exceed list end", 400)
		return
	}
	writeJson(res, rsrcs)
}

func resource(res http.ResponseWriter, req *http.Request) {
	if req.Method != "GET" {
		res.WriteHeader(405)
		return
	}
	if !strings.Contains(req.URL.String(), "?") {
		res.WriteHeader(400)
		return
	}
	idstr, ok := req.URL.Query()["id"]
	if !ok {
		res.WriteHeader(400)
		return
	}
	rsrc, err := client.GetFile(req.Context(), idstr[0])
	// TODO id doesn't exist
	if err != nil {
		res.WriteHeader(500)
		log.Println("error finding resource", err)
	}
	writeJson(res, rsrc)
}

func resourceTags(res http.ResponseWriter, req *http.Request) {
	if req.Method != "POST" {
		res.WriteHeader(405)
		return
	}
	var tc TagChange
	dec := json.NewDecoder(req.Body)
	err := dec.Decode(&tc)
	if err != nil {
		res.WriteHeader(400)
		return
	}
	var addtags, deltags model.TagSet
	addtags.FillFromString(tc.AddTags)
	deltags.FillFromString(tc.DelTags)
	err = client.ChangeTags(req.Context(), addtags, deltags, tc.ResourceId)
	if err != nil {
		res.WriteHeader(500)
		return
	}
	rsc, err := client.GetFile(req.Context(), tc.ResourceId)
	if err != nil {
		res.WriteHeader(500)
		return
	}
	writeJson(res, rsc)
}

func settingsPage(res http.ResponseWriter, req *http.Request) {
	if req.Method == "GET" {
		us := req.Context().Value(model.CtxkeyUserSettings(0)).(model.UserSettings)
		err := templates.ExecuteTemplate(res, "settings.gohtml", struct {
			PageMeta     model.PageMeta
			UserSettings model.UserSettings
		}{
			model.PageMeta{
				Title: "Settings",
			},
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
	var ust model.UserSettings
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
	res.Header().Add("location", config.Global.UrlBase+"/site/settings")
	res.WriteHeader(303)
}

func servefile(res http.ResponseWriter, req *http.Request) {
	id := req.URL.Path[len("/files/"):]
	bts, err := client.GetBytes(req.Context(), id)
	if err != nil {
		log.Println(err)
		http.Error(res, "Server error", 500)
		return
	}
	res.Write(bts)
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
		var userSettings model.UserSettings
		cookie, err := req.Cookie("settings")
		if err == http.ErrNoCookie {
			userSettings = model.DefaultUserSettings
			goto gonext
		}
		err = userSettings.FromCookieString(cookie.Value)
		if err != nil {
			log.Println("Could not get userSettings:", err)
		}
	gonext:
		next.ServeHTTP(res, req.WithContext(context.WithValue(ctx, model.CtxkeyUserSettings(0), userSettings)))
	})
}

func main() {
	// Setup Context
	var ctx = context.Background()

	// Load config
	config.Load()

	// Load Templates
	templates = template.Must(template.New("").Funcs(map[string]any{
		"hasPrefix":   strings.HasPrefix,
		"getBaseUrl":  func() string { return config.Global.UrlBase },
		"stringifyTS": func(ts model.TagSet) string { return ts.String() },
	}).ParseGlob("./templates/*.gohtml"))

	// Load database connection
	dbc, err := db.ConnectDatabases(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer dbc.Close(ctx)
	client = dbc

	server := http.NewServeMux()

	statfs := http.FileServer(http.Dir("./dist"))

	server.HandleFunc("/", landingPage)
	server.Handle("/public/", http.StripPrefix("/public/", statfs))
	server.HandleFunc("/files/", servefile)
	server.HandleFunc("/site/upload", multiuploadPage)
	server.HandleFunc("/api/query", query)
	server.HandleFunc("/api/resource", resource)
	server.HandleFunc("/api/resource/tags", resourceTags)
	server.HandleFunc("/site/settings", settingsPage)

	log.Fatal(http.ListenAndServe(":8080", addContext(ctx, attachUserSettings(http.StripPrefix(config.Global.UrlBase, server)))))
}
