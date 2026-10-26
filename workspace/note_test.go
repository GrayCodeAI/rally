package workspace

import "testing"

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
