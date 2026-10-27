package workspace

import (
	"sync"
	"testing"
)

func TestPostNamedReusesChannelAndAgent(t *testing.T) {
	s := &Store{Dir: t.TempDir()}
	first, err := s.PostNamed("engineering", "Reflect", "", "started")
	if err != nil {
		t.Fatal(err)
	}
	second, err := s.PostNamed("engineering", "Reflect", "", "verified")
	if err != nil {
		t.Fatal(err)
	}
	if first.ChannelID != second.ChannelID || first.ActorID != second.ActorID {
		t.Fatal("note created duplicate channel or agent")
	}
	messages, err := s.Messages(first.ChannelID)
	if err != nil || len(messages) != 2 {
		t.Fatalf("messages = %d, %v", len(messages), err)
	}
}

func TestPostNamedAcrossStoreInstances(t *testing.T) {
	dir := t.TempDir()
	const count = 24
	results := make(chan Message, count)
	errors := make(chan error, count)
	var wg sync.WaitGroup
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			message, err := (&Store{Dir: dir}).PostNamed("engineering", "Reflect", "", "update")
			if err != nil {
				errors <- err
				return
			}
			results <- message
		}()
	}
	wg.Wait()
	close(results)
	close(errors)
	for err := range errors {
		t.Fatal(err)
	}
	var first Message
	for message := range results {
		if first.ID == "" {
			first = message
		}
		if message.ChannelID != first.ChannelID || message.ActorID != first.ActorID {
			t.Fatal("concurrent notes used different channel or member IDs")
		}
	}
	store := &Store{Dir: dir}
	channels, err := store.Channels()
	if err != nil || len(channels) != 1 {
		t.Fatalf("channels = %d, %v", len(channels), err)
	}
	members, err := store.Members()
	if err != nil || len(members) != 1 {
		t.Fatalf("members = %d, %v", len(members), err)
	}
}
