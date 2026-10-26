package workspace

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type Member struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Kind      string    `json:"kind"`
	CreatedAt time.Time `json:"created_at"`
}

type Channel struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type Message struct {
	ID        string    `json:"id"`
	ChannelID string    `json:"channel_id"`
	ActorID   string    `json:"actor_id"`
	ThreadID  string    `json:"thread_id,omitempty"`
	TicketID  string    `json:"ticket_id,omitempty"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}

type Store struct {
	Dir string
	mu  sync.Mutex
}

// PostNamed posts a local workflow update, creating the named channel and
// agent member on first use. Existing IDs remain stable across CLI invocations.
func (s *Store) PostNamed(channelName, actorName, ticketID, body string) (Message, error) {
	channelName, actorName = strings.TrimSpace(channelName), strings.TrimSpace(actorName)
	if channelName == "" || actorName == "" {
		return Message{}, errors.New("channel and actor names are required")
	}
	channels, err := s.Channels()
	if err != nil {
		return Message{}, err
	}
	var channelID string
	for _, channel := range channels {
		if channel.Name == channelName {
			channelID = channel.ID
			break
		}
	}
	if channelID == "" {
		channel, err := s.CreateChannel(channelName)
		if err != nil {
			return Message{}, err
		}
		channelID = channel.ID
	}
	members, err := s.Members()
	if err != nil {
		return Message{}, err
	}
	var actorID string
	for _, member := range members {
		if member.Name == actorName && member.Kind == "agent" {
			actorID = member.ID
			break
		}
	}
	if actorID == "" {
		member, err := s.CreateMember(actorName, "agent")
		if err != nil {
			return Message{}, err
		}
		actorID = member.ID
	}
	return s.Post(channelID, actorID, "", ticketID, body)
}

func (s *Store) CreateMember(name, kind string) (Member, error) {
	name = strings.TrimSpace(name)
	if name == "" || (kind != "human" && kind != "agent") {
		return Member{}, errors.New("name and kind (human or agent) are required")
	}
	id, err := randomID()
	if err != nil {
		return Member{}, err
	}
	m := Member{ID: id, Name: name, Kind: kind, CreatedAt: time.Now().UTC()}
	return m, s.write("members", m.ID, m)
}

func (s *Store) CreateChannel(name string) (Channel, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Channel{}, errors.New("channel name is required")
	}
	id, err := randomID()
	if err != nil {
		return Channel{}, err
	}
	c := Channel{ID: id, Name: name, CreatedAt: time.Now().UTC()}
	return c, s.write("channels", c.ID, c)
}

func (s *Store) Post(channelID, actorID, threadID, ticketID, body string) (Message, error) {
	if !validID(channelID) || !validID(actorID) || strings.TrimSpace(body) == "" {
		return Message{}, errors.New("channel_id, actor_id, and body are required")
	}
	if ticketID != "" && !validID(ticketID) {
		return Message{}, errors.New("ticket_id must be a Resolve ticket ID")
	}
	if _, err := s.read("members", actorID, &Member{}); err != nil {
		return Message{}, fmt.Errorf("actor: %w", err)
	}
	if _, err := s.read("channels", channelID, &Channel{}); err != nil {
		return Message{}, fmt.Errorf("channel: %w", err)
	}
	if threadID != "" {
		var parent Message
		if _, err := s.read("messages", threadID, &parent); err != nil {
			return Message{}, fmt.Errorf("thread: %w", err)
		}
		if parent.ChannelID != channelID {
			return Message{}, errors.New("thread belongs to a different channel")
		}
	}
	id, err := randomID()
	if err != nil {
		return Message{}, err
	}
	m := Message{ID: id, ChannelID: channelID, ActorID: actorID, ThreadID: threadID, TicketID: ticketID, Body: strings.TrimSpace(body), CreatedAt: time.Now().UTC()}
	return m, s.write("messages", m.ID, m)
}

func (s *Store) Members() ([]Member, error) {
	var out []Member
	err := s.walk("members", func(data []byte) error {
		var m Member
		if err := json.Unmarshal(data, &m); err != nil {
			return err
		}
		out = append(out, m)
		return nil
	})
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, err
}

func (s *Store) Channels() ([]Channel, error) {
	var out []Channel
	err := s.walk("channels", func(data []byte) error {
		var c Channel
		if err := json.Unmarshal(data, &c); err != nil {
			return err
		}
		out = append(out, c)
		return nil
	})
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, err
}

func (s *Store) Messages(channelID string) ([]Message, error) {
	if !validID(channelID) {
		return nil, errors.New("valid channel_id is required")
	}
	var out []Message
	err := s.walk("messages", func(data []byte) error {
		var m Message
		if err := json.Unmarshal(data, &m); err != nil {
			return err
		}
		if m.ChannelID == channelID {
			out = append(out, m)
		}
		return nil
	})
	sort.Slice(out, func(i, j int) bool {
		if out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].ID < out[j].ID
		}
		return out[i].CreatedAt.Before(out[j].CreatedAt)
	})
	return out, err
}

func (s *Store) write(kind, id string, value any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	dir := filepath.Join(s.Dir, kind)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	name := filepath.Join(dir, id+".json")
	f, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Write(append(data, '\n')); err != nil {
		os.Remove(name)
		return err
	}
	if err := f.Sync(); err != nil {
		os.Remove(name)
		return err
	}
	return nil
}

func (s *Store) read(kind, id string, target any) (bool, error) {
	if !validID(id) {
		return false, errors.New("invalid ID")
	}
	data, err := os.ReadFile(filepath.Join(s.Dir, kind, id+".json"))
	if err != nil {
		return false, err
	}
	return true, json.Unmarshal(data, target)
}

func (s *Store) walk(kind string, each func([]byte) error) error {
	dir := filepath.Join(s.Dir, kind)
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") || !validID(strings.TrimSuffix(entry.Name(), ".json")) {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return err
		}
		if err := each(data); err != nil {
			return err
		}
	}
	return nil
}

func randomID() (string, error) {
	var b [16]byte
	_, err := rand.Read(b[:])
	return hex.EncodeToString(b[:]), err
}

func validID(id string) bool {
	if len(id) != 32 || id != strings.ToLower(id) {
		return false
	}
	_, err := hex.DecodeString(id)
	return err == nil
}
