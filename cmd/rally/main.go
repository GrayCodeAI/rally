package main

import (
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
	if len(args) == 0 || args[0] != "serve" {
		return errors.New("use rally serve [--data dir] [--listen 127.0.0.1:8765]")
	}
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	data := fs.String("data", defaultDataDir(), "workspace data directory")
	listen := fs.String("listen", "127.0.0.1:8765", "loopback listener")
	if err := fs.Parse(args[1:]); err != nil {
		return err
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
