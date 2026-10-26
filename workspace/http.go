package workspace

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
)

func Handler(store *Store) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/members", func(w http.ResponseWriter, r *http.Request) {
		var input struct{ Name, Kind string }
		if err := decode(r, &input); err != nil {
			fail(w, err, http.StatusBadRequest)
			return
		}
		value, err := store.CreateMember(input.Name, input.Kind)
		respond(w, value, err, http.StatusCreated)
	})
	mux.HandleFunc("GET /v1/members", func(w http.ResponseWriter, r *http.Request) {
		value, err := store.Members()
		respond(w, value, err, http.StatusOK)
	})
	mux.HandleFunc("POST /v1/channels", func(w http.ResponseWriter, r *http.Request) {
		var input struct{ Name string }
		if err := decode(r, &input); err != nil {
			fail(w, err, http.StatusBadRequest)
			return
		}
		value, err := store.CreateChannel(input.Name)
		respond(w, value, err, http.StatusCreated)
	})
	mux.HandleFunc("GET /v1/channels", func(w http.ResponseWriter, r *http.Request) {
		value, err := store.Channels()
		respond(w, value, err, http.StatusOK)
	})
	mux.HandleFunc("POST /v1/messages", func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			ChannelID string `json:"channel_id"`
			ActorID   string `json:"actor_id"`
			ThreadID  string `json:"thread_id"`
			TicketID  string `json:"ticket_id"`
			Body      string `json:"body"`
		}
		if err := decode(r, &input); err != nil {
			fail(w, err, http.StatusBadRequest)
			return
		}
		value, err := store.Post(input.ChannelID, input.ActorID, input.ThreadID, input.TicketID, input.Body)
		respond(w, value, err, http.StatusCreated)
	})
	mux.HandleFunc("GET /v1/messages", func(w http.ResponseWriter, r *http.Request) {
		value, err := store.Messages(r.URL.Query().Get("channel_id"))
		respond(w, value, err, http.StatusOK)
	})
	return mux
}

func decode(r *http.Request, target any) error {
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		return errors.New("Content-Type must be application/json")
	}
	d := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	d.DisallowUnknownFields()
	if err := d.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := d.Decode(&extra); err != io.EOF {
		return errors.New("expected one JSON object")
	}
	return nil
}

func respond(w http.ResponseWriter, value any, err error, success int) {
	if err != nil {
		fail(w, err, http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(success)
	_ = json.NewEncoder(w).Encode(value)
}

func fail(w http.ResponseWriter, err error, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
