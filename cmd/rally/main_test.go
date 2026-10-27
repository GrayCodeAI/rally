package main

import (
	"encoding/json"
	"io"
	"os"
	"testing"

	"github.com/GrayCodeAI/rally/workspace"
)

func TestListNamedChannel(t *testing.T) {
	dir := t.TempDir()
	store := &workspace.Store{Dir: dir}
	first, err := store.PostNamed("project", "Reflect", "", "started")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.PostNamed("other", "Reflect", "", "elsewhere"); err != nil {
		t.Fatal(err)
	}
	read, write, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdout
	os.Stdout = write
	t.Cleanup(func() { os.Stdout = old })
	if err := run([]string{"list", "--data", dir, "--channel", "project"}); err != nil {
		t.Fatal(err)
	}
	write.Close()
	data, err := io.ReadAll(read)
	if err != nil {
		t.Fatal(err)
	}
	var messages []workspace.Message
	if err := json.Unmarshal(data, &messages); err != nil {
		t.Fatal(err)
	}
	if len(messages) != 1 || messages[0].ID != first.ID {
		t.Fatalf("messages = %+v", messages)
	}
}
