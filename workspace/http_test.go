package workspace

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPersistentConversationAndTicketLink(t *testing.T) {
	dir := t.TempDir()
	h := Handler(&Store{Dir: dir})
	var person Member
	request(t, h, "POST", "/v1/members", map[string]any{"name": "Alice", "kind": "human"}, 201, &person)
	var agent Member
	request(t, h, "POST", "/v1/members", map[string]any{"name": "Builder", "kind": "agent"}, 201, &agent)
	var channel Channel
	request(t, h, "POST", "/v1/channels", map[string]any{"name": "engineering"}, 201, &channel)
	ticketID := "0123456789abcdef0123456789abcdef"
	var root Message
	request(t, h, "POST", "/v1/messages", map[string]any{"channel_id": channel.ID, "actor_id": person.ID, "ticket_id": ticketID, "body": "Please investigate"}, 201, &root)
	var reply Message
	request(t, h, "POST", "/v1/messages", map[string]any{"channel_id": channel.ID, "actor_id": agent.ID, "thread_id": root.ID, "ticket_id": ticketID, "body": "I found the failure"}, 201, &reply)
	if reply.ThreadID != root.ID || reply.TicketID != ticketID {
		t.Fatalf("missing links: %#v", reply)
	}
	var messages []Message
	request(t, Handler(&Store{Dir: dir}), "GET", "/v1/messages?channel_id="+channel.ID, nil, 200, &messages)
	if len(messages) != 2 || messages[0].ID != root.ID || messages[1].ID != reply.ID {
		t.Fatalf("conversation did not survive store restart: %#v", messages)
	}
	var bad map[string]any
	request(t, h, "POST", "/v1/messages", map[string]any{"channel_id": channel.ID, "actor_id": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "body": "Spoofed"}, 400, &bad)
	request(t, h, "POST", "/v1/messages", map[string]any{"channel_id": channel.ID, "actor_id": person.ID, "thread_id": "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", "body": "Broken thread"}, 400, &bad)
}

func request(t *testing.T, h http.Handler, method, path string, body any, want int, out any) {
	t.Helper()
	var raw []byte
	if body != nil {
		var err error
		raw, err = json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
	}
	r := httptest.NewRequest(method, path, bytes.NewReader(raw))
	if body != nil {
		r.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != want {
		t.Fatalf("%s %s: got %d, want %d: %s", method, path, w.Code, want, w.Body.String())
	}
	if err := json.Unmarshal(w.Body.Bytes(), out); err != nil {
		t.Fatal(err)
	}
}
