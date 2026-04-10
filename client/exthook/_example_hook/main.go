// Example stdin hook: append each JSON line to a log file for spike testing.
//
// Build: go build -o marchat-hook-log ./client/exthook/_example_hook
// Run client from repo root: go run ./client  (not go run .)
// Env: MARCHAT_CLIENT_HOOK_RECEIVE=C:\full\path\marchat-hook-log.exe (and/or MARCHAT_CLIENT_HOOK_SEND)
//
// Default log: $TEMP/marchat-client-hook.log (Windows) or /tmp/marchat-client-hook.log
package main

import (
	"bufio"
	"io"
	"log"
	"os"
	"path/filepath"
)

func main() {
	logPath := os.Getenv("MARCHAT_HOOK_LOG")
	if logPath == "" {
		if d := os.Getenv("TEMP"); d != "" {
			logPath = filepath.Join(d, "marchat-client-hook.log")
		} else if d := os.Getenv("TMPDIR"); d != "" {
			logPath = filepath.Join(d, "marchat-client-hook.log")
		} else {
			logPath = filepath.Join(os.TempDir(), "marchat-client-hook.log")
		}
	}
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	r := bufio.NewReader(os.Stdin)
	line, err := r.ReadBytes('\n')
	if err != nil && err != io.EOF {
		log.Fatal(err)
	}
	if len(line) == 0 {
		return
	}
	if _, err := f.Write(line); err != nil {
		log.Fatal(err)
	}
	if line[len(line)-1] != '\n' {
		_, _ = f.Write([]byte{'\n'})
	}
}
