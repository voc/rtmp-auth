package http

import (
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/gorilla/csrf"
	"github.com/voc/rtmp-auth/storage"
	"github.com/voc/rtmp-auth/store"
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
func parseExpiry(str string) *int64 {
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

func PublishHandler(store *store.Store) handleFunc {
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

		success, id := store.Auth(app, name, auth)
		if !success {
			log.Printf("Publish %s %s/%s unauthorized\n", id, app, name)
			http.Error(w, "401 Unauthorized", http.StatusUnauthorized)
			return
		}

		store.SetActive(id)
		log.Printf("Publish %s %s/%s ok\n", id, app, name)
	}
}

func UnpublishHandler(store *store.Store) handleFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			log.Println("Failed to parse unpublish data:", err)
			http.Error(w, "401 Unauthorized", http.StatusUnauthorized)
			return
		}

		app := r.PostForm.Get("app")
		name := r.PostForm.Get("name")

		store.SetInactive(app, name)
		log.Printf("Unpublish %s/%s ok\n", app, name)
	}
}

func FormHandler(store *store.Store) handleFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := TemplateData{
			Store:        store.Get(),
			CsrfTemplate: csrf.TemplateField(r),
		}
		err := templates.ExecuteTemplate(w, "form.html", data)
		if err != nil {
			log.Println("Template failed", err)
		}
	}
}

func AddHandler(store *store.Store, prefix string) handleFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var errs []error

		expiry := parseExpiry(r.PostFormValue("auth_expire"))
		if expiry == nil {
			errs = append(errs, fmt.Errorf("invalid auth expiry: '%v'", r.PostFormValue("auth_expire")))
		}

		name := r.PostFormValue("name")
		if len(name) == 0 {
			errs = append(errs, fmt.Errorf("stream name must be set"))
		}

		// TODO: more validation
		if len(errs) == 0 {
			stream := &storage.Stream{
				Name:        name,
				Application: r.PostFormValue("application"),
				AuthKey:     r.PostFormValue("auth_key"),
				AuthExpire:  *expiry,
				Notes:       r.PostFormValue("notes"),
			}

			err := store.AddStream(stream)
			log.Println("store add", stream, store.State)
			if err != nil {
				errs = append(errs, fmt.Errorf("failed to add stream: %w", err))
			} else {
				http.Redirect(w, r, prefix, http.StatusSeeOther)
			}
		}

		data := TemplateData{
			Store:        store.Get(),
			CsrfTemplate: csrf.TemplateField(r),
			Errors:       errs,
		}
		err := templates.ExecuteTemplate(w, "form.html", data)
		if err != nil {
			log.Println("Template failed", err)
		}
	}
}

func RemoveHandler(store *store.Store, prefix string) handleFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var errs []error
		id := r.PostFormValue("id")

		err := store.RemoveStream(id)
		if err != nil {
			log.Println(err)
			errs = append(errs, fmt.Errorf("failed to remove stream: %w", err))
			data := TemplateData{
				Store:        store.Get(),
				CsrfTemplate: csrf.TemplateField(r),
				Errors:       errs,
			}
			err = templates.ExecuteTemplate(w, "form.html", data)
			if err != nil {
				log.Println("Template failed", err)
			}
		} else {
			http.Redirect(w, r, prefix, http.StatusSeeOther)
		}
	}
}

func BlockHandler(store *store.Store, prefix string) handleFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var errs []error
		id := r.PostFormValue("id")
		newState := false
		action := "unblock"
		state, _ := strconv.ParseBool(r.PostFormValue("blocked"))

		if !state {
			newState = true
			action = "block"
		}

		// Get Application/Name for stream id
		var app, name string
		for _, stream := range store.State.Streams {
			if stream.Id == id {
				app = stream.Application
				name = stream.Name
			}
		}

		err := store.SetBlocked(id, newState)
		log.Printf("%ved Stream %v (%v/%v)", action, id, app, name)
		if err != nil {
			log.Println(err)
			errs = append(errs, fmt.Errorf("failed to %v stream %v (%v/%v)", action, id, app, name))

			data := TemplateData{
				Store:        store.Get(),
				CsrfTemplate: csrf.TemplateField(r),
				Errors:       errs,
			}
			err = templates.ExecuteTemplate(w, "form.html", data)
			if err != nil {
				log.Println("Template failed", err)
			}
		} else {
			http.Redirect(w, r, prefix, http.StatusSeeOther)
		}
	}
}
