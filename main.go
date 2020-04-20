package main

import (
    "context"
    "flag"
    "fmt"
    "log"
    "net/http"
    "os"
    "os/signal"
    "regexp"
    "strconv"
    "strings"
    "time"

    "github.com/gorilla/csrf"
    "github.com/gorilla/mux"
    "github.com/rakyll/statik/fs"

    "github.com/voc/rtmp-auth/storage"
    _ "github.com/voc/rtmp-auth/statik"
)

type handleFunc func(http.ResponseWriter, *http.Request)

var durationRegex = regexp.MustCompile(`P([\d\.]+Y)?([\d\.]+M)?([\d\.]+D)?T?([\d\.]+H)?([\d\.]+M)?([\d\.]+?S)?`)

func parseDurationPart(value string, unit time.Duration) time.Duration {
   if len(value) != 0 {
       if parsed, err := strconv.ParseFloat(value[:len(value)-1], 64); err == nil {
           return time.Duration(float64(unit) * parsed)
       }
   }
   return 0
}

// Parse expiration time
func ParseExpiry(str string) *int64 {
    // Allow empty string for "never"
    if str == "" {
        never := int64(-1)
        return &never
    }

    // Try to parse as ISO8601 duration
    matches := durationRegex.FindStringSubmatch(str)
    if matches != nil {
        years := parseDurationPart(matches[1], time.Hour*24*365)
        months := parseDurationPart(matches[2], time.Hour*24*30)
        days := parseDurationPart(matches[3], time.Hour*24)
        hours := parseDurationPart(matches[4], time.Hour)
        minutes := parseDurationPart(matches[5], time.Second*60)
        seconds := parseDurationPart(matches[6], time.Second)
        d := time.Duration(years + months + days + hours + minutes + seconds)
        if d == 0 {
            return nil
        }

        expiry := time.Now().Add(d).Unix()
        return &expiry
    }

    // Try to parse as absolute time
    t, err := time.Parse(time.RFC3339, str)
    if err != nil {
        return nil
    }
    expiry := t.Unix()
    return &expiry
}

func PublishHandler(store *Store) handleFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        err := r.ParseForm()
        if err != nil {
            log.Println("Failed to parse publish data:", err)
            http.Error(w, "401 Unauthorized", http.StatusUnauthorized)
            return
        }

        app := r.PostForm.Get("app")
        name := r.PostForm.Get("name")
        auth := r.PostForm.Get("auth")

        log.Printf("publish %s/%s auth: '%s'\n", app, name, auth)

        success := store.Auth(app, name, auth)
        if !success {
            log.Printf("Publish %s/%s unauthorized\n", app, name)
            http.Error(w, "401 Unauthorized", http.StatusUnauthorized)
            return
        }

        store.SetActive(app, name, true)
        log.Printf("Publish %s/%s ok\n", app, name)
    }
}

func UnpublishHandler(store *Store) handleFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        err := r.ParseForm()
        if err != nil {
            log.Println("Failed to parse unpublish data:", err)
            http.Error(w, "401 Unauthorized", http.StatusUnauthorized)
            return
        }

        app := r.PostForm.Get("app")
        name := r.PostForm.Get("name")

        store.SetActive(app, name, false)
        log.Printf("Unpublish %s/%s ok\n", app, name)
    }
}

func FormHandler(store *Store) handleFunc {
    return func(w http.ResponseWriter, r *http.Request) {        data := TemplateData{
            Store: store.Get(),
            CsrfTemplate: csrf.TemplateField(r),
        }
        err := templates.ExecuteTemplate(w, "form.html", data)
        if err != nil {
            log.Println("Template failed", err)
        }
    }
}

func AddHandler(store *Store) handleFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        var errs []error

        expiry := ParseExpiry(r.PostFormValue("auth_expire"))
        if expiry == nil {
            errs = append(errs, fmt.Errorf("Invalid auth expiry: '%v'", r.PostFormValue("auth_expire")))
        }

        name := r.PostFormValue("name")
        if len(name) == 0 {
            errs = append(errs, fmt.Errorf("Stream name must be set"))
        }

        // TODO: more validation
        if len(errs) == 0 {
            stream := &storage.Stream{
                Name: name,
                Application: r.PostFormValue("application"),
                AuthKey: r.PostFormValue("auth_key"),
                AuthExpire: *expiry,
            }

            store.AddStream(stream)
            log.Println("store add", stream, store.State)
        }

        data := TemplateData{
            Store: store.Get(),
            CsrfTemplate: csrf.TemplateField(r),
            Errors: errs,
        }
        err := templates.ExecuteTemplate(w, "form.html", data)
        if err != nil {
            log.Println("Template failed", err)
        }
    }
}

func RemoveHandler(store *Store) handleFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        var errs []error
        id := r.PostFormValue("id")
        err := store.RemoveStream(id)
        if err != nil {
            log.Println(err)
            errs = append(errs, fmt.Errorf("Failed to remove stream: %v", err))
        }
        data := TemplateData{
            Store: store.Get(),
            CsrfTemplate: csrf.TemplateField(r),
            Errors: errs,
        }
        err = templates.ExecuteTemplate(w, "form.html", data)
        if err != nil {
            log.Println("Template failed", err)
        }
    }
}

func main() {
    var path = flag.String("store", "store.db", "Path to store file")
    var apps = flag.String("app", "stream", "Comma separated list of RTMP applications")
    var apiAddr = flag.String("apiAddr", "localhost:8080", "API bind address")
    var frontendAddr = flag.String("frontendAddr", "localhost:8082", "Frontend bind address")
    var insecure = flag.Bool("insecure", false, "Set to allow non-secure CSRF cookie")
    flag.Parse()

    store, err := NewStore(*path, strings.Split(*apps, ","))
    if err != nil {
        log.Fatal("noo", err)
    }

    statikFS, err := fs.New()
    if err != nil {
        log.Fatal(err)
    }

    CSRF := csrf.Protect([]byte("32-byte-long-auth-key"), csrf.Secure(!*insecure))

    api := mux.NewRouter()
    api.Path("/publish").Methods("POST").HandlerFunc(PublishHandler(store));
    api.Path("/unpublish").Methods("POST").HandlerFunc(UnpublishHandler(store));

    frontend := mux.NewRouter()
    frontend.Path("/").Methods("GET").HandlerFunc(FormHandler(store));
    frontend.Path("/add").Methods("POST").HandlerFunc(AddHandler(store));
    frontend.Path("/remove").Methods("POST").HandlerFunc(RemoveHandler(store));
    frontend.PathPrefix("/public/").Handler(
        http.StripPrefix("/public/", http.FileServer(statikFS)));


    apiServer := &http.Server{
        Handler: api,
        Addr:    *apiAddr,
        WriteTimeout: 15 * time.Second,
        ReadTimeout:  15 * time.Second,
    }

    frontendServer := &http.Server{
        Handler: CSRF(frontend),
        Addr:    *frontendAddr,
        WriteTimeout: 15 * time.Second,
        ReadTimeout:  15 * time.Second,
    }

    // Periodically expire old streams
    ticker := time.NewTicker(10 * time.Second)
    stopPolling := make(chan struct{})
    go func(){
        for {
            select {
            case <-stopPolling:
                return
            case <-ticker.C:
                store.Expire()
            }
        }
    }()

    // Run http servers
    go func() {
        log.Println("API Listening on", apiServer.Addr)
        if err := apiServer.ListenAndServe(); err != nil {
            log.Println(err)
        }
    }()
    go func() {
        log.Println("Frontend Listening on", frontendServer.Addr)
        if err := frontendServer.ListenAndServe(); err != nil {
            log.Println(err)
        }
    }()

    // Handle signals
    c := make(chan os.Signal, 1)
    signal.Notify(c, os.Interrupt)
    <-c

    ctx, cancel := context.WithTimeout(context.Background(), 100 * time.Millisecond)
    defer cancel()

    // Shut everything down
    close(stopPolling)
    go apiServer.Shutdown(ctx)
    go frontendServer.Shutdown(ctx)

    // Wait until timeout
    log.Println("Shutting down")
    <-ctx.Done()
}
