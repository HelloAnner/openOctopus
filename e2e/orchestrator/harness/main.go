package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/anner/openoctopus/internal/orchestrator"
)

var messagePattern = regexp.MustCompile(`## message: msg-(\d{6})`)

func main() {
	if len(os.Args) < 2 {
		fail(errors.New("missing command"))
	}
	var err error
	switch os.Args[1] {
	case "tick":
		err = runTick(os.Args[2:])
	case "append-human-message":
		err = runAppendHumanMessage(os.Args[2:])
	case "write-conclusion":
		err = runWriteConclusion(os.Args[2:])
	default:
		err = fmt.Errorf("unsupported command: %s", os.Args[1])
	}
	if err != nil {
		fail(err)
	}
}

func runTick(args []string) error {
	flags := flag.NewFlagSet("tick", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	sessionDir := flags.String("session-dir", "", "session dir")
	if err := flags.Parse(args); err != nil {
		return err
	}
	result, err := orchestrator.NewEngine(*sessionDir).Tick()
	if err != nil {
		return err
	}
	return writeJSON(result)
}

func runAppendHumanMessage(args []string) error {
	flags := flag.NewFlagSet("append-human-message", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	sessionDir := flags.String("session-dir", "", "session dir")
	source := flags.String("source", "user", "source")
	message := flags.String("message", "", "message")
	if err := flags.Parse(args); err != nil {
		return err
	}
	path := filepath.Join(*sessionDir, "planner", "human_messages.md")
	content := "# Human Messages\n"
	if existing, err := os.ReadFile(path); err == nil {
		current := string(existing)
		if !strings.Contains(current, "Initialized by session 001.") && strings.TrimSpace(current) != "" {
			content = strings.TrimRight(current, "\n") + "\n"
		}
	}
	nextID := nextMessageID(content)
	block := fmt.Sprintf("\n## message: %s\n- message_id: %s\n- source: %s\n- created_at: %s\n\n### content\n%s\n", nextID, nextID, *source, time.Now().UTC().Format(time.RFC3339), *message)
	return os.WriteFile(path, []byte(strings.TrimRight(content, "\n")+block), 0o644)
}

func runWriteConclusion(args []string) error {
	flags := flag.NewFlagSet("write-conclusion", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	sessionDir := flags.String("session-dir", "", "session dir")
	roleID := flags.String("role-id", "", "role id")
	stageID := flags.String("stage-id", "", "stage id")
	taskID := flags.String("task-id", "", "task id")
	status := flags.String("status", "", "status")
	summary := flags.String("summary", "", "summary")
	if err := flags.Parse(args); err != nil {
		return err
	}
	roleDir := filepath.Join(*sessionDir, "roles", *roleID)
	if err := os.MkdirAll(roleDir, 0o755); err != nil {
		return err
	}
	content := fmt.Sprintf("# Role Conclusion\n\n- role_id: %s\n- stage_id: %s\n- task_id: %s\n- status: %s\n- summary: %s\n- output_refs: \n- updated_at: %s\n", *roleID, *stageID, *taskID, *status, *summary, time.Now().UTC().Format(time.RFC3339))
	return os.WriteFile(filepath.Join(roleDir, "conclusion.md"), []byte(content), 0o644)
}

func nextMessageID(content string) string {
	matches := messagePattern.FindAllStringSubmatch(content, -1)
	if len(matches) == 0 {
		return "msg-000001"
	}
	last := matches[len(matches)-1][1]
	value, err := strconv.Atoi(last)
	if err != nil {
		return "msg-000001"
	}
	return fmt.Sprintf("msg-%06d", value+1)
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
