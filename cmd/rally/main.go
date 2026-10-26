package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"

	"github.com/GrayCodeAI/rally/workspace"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "rally:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 || (args[0] != "serve" && args[0] != "note") {
		return errors.New("use rally serve or rally note")
	}
	fs := flag.NewFlagSet(args[0], flag.ContinueOnError)
	data := fs.String("data", defaultDataDir(), "workspace data directory")
	listen := fs.String("listen", "127.0.0.1:8765", "loopback listener")
	channel := fs.String("channel", "engineering", "channel name for a local note")
	actor := fs.String("actor", "Reflect", "agent member name for a local note")
	ticket := fs.String("ticket", "", "optional Resolve ticket ID")
	text := fs.String("text", "", "note body")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if args[0] == "note" {
		message, err := (&workspace.Store{Dir: *data}).PostNamed(*channel, *actor, *ticket, *text)
		if err != nil {
			return err
		}
		return json.NewEncoder(os.Stdout).Encode(message)
	}
	host, _, err := net.SplitHostPort(*listen)
	if err != nil {
		return err
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return errors.New("Rally only listens on a loopback IP until authentication is implemented")
	}
	fmt.Fprintln(os.Stderr, "Rally listening on", *listen)
	return http.ListenAndServe(*listen, workspace.Handler(&workspace.Store{Dir: *data}))
}

func defaultDataDir() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ".rally"
	}
	return filepath.Join(dir, "rally", "workspace")
}
