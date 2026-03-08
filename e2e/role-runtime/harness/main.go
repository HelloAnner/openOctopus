package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/anner/openoctopus/internal/roleruntime"
)

func main() {
	if len(os.Args) < 2 {
		fail(errors.New("missing command"))
	}
	var err error
	switch os.Args[1] {
	case "tick-role":
		err = runTickRole(os.Args[2:])
	case "tick-all":
		err = runTickAll(os.Args[2:])
	case "write-reset":
		err = runWriteReset(os.Args[2:])
	default:
		err = fmt.Errorf("unsupported command: %s", os.Args[1])
	}
	if err != nil {
		fail(err)
	}
}

func runTickRole(args []string) error {
	flags := flag.NewFlagSet("tick-role", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	sessionDir := flags.String("session-dir", "", "session dir")
	roleID := flags.String("role-id", "", "role id")
	if err := flags.Parse(args); err != nil {
		return err
	}
	result, err := roleruntime.NewEngine(*sessionDir).TickRole(*roleID)
	if err != nil {
		return err
	}
	return writeJSON(result)
}

func runTickAll(args []string) error {
	flags := flag.NewFlagSet("tick-all", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	sessionDir := flags.String("session-dir", "", "session dir")
	if err := flags.Parse(args); err != nil {
		return err
	}
	result, err := roleruntime.NewEngine(*sessionDir).TickAll()
	if err != nil {
		return err
	}
	return writeJSON(result)
}

func runWriteReset(args []string) error {
	flags := flag.NewFlagSet("write-reset", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	sessionDir := flags.String("session-dir", "", "session dir")
	roleID := flags.String("role-id", "", "role id")
	reason := flags.String("reason", "", "reason")
	requestedBy := flags.String("requested-by", "test", "requested by")
	if err := flags.Parse(args); err != nil {
		return err
	}
	roleDir := filepath.Join(*sessionDir, "roles", *roleID)
	if err := os.MkdirAll(roleDir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(roleDir, "session.reset.md")
	content := fmt.Sprintf("# Session Reset\n\n- session_generation: 1\n- status: REQUESTED\n- requested_by: %s\n- reason: %s\n- requested_at: %s\n- applied_at: \n- last_cleared_task_id: \n", *requestedBy, *reason, time.Now().UTC().Format(time.RFC3339))
	return os.WriteFile(path, []byte(content), 0o644)
}

func writeJSON(value any) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(os.Stdout, "%s\n", payload)
	return err
}

func fail(err error) {
	_, _ = fmt.Fprintln(os.Stderr, err.Error())
	os.Exit(1)
}
