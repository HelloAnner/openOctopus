package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/anner/openoctopus/internal/eventbus"
)

func main() {
	if len(os.Args) < 2 {
		fail(errors.New("missing command"))
	}
	var err error
	switch os.Args[1] {
	case "bootstrap":
		err = runBootstrap(os.Args[2:])
	case "list":
		err = runList(os.Args[2:])
	case "acquire-lock":
		err = runAcquireLock(os.Args[2:])
	case "renew-lock":
		err = runRenewLock(os.Args[2:])
	case "release-lock":
		err = runReleaseLock(os.Args[2:])
	case "append":
		err = runAppend(os.Args[2:])
	case "commit-offset":
		err = runCommitOffset(os.Args[2:])
	case "request-interrupt":
		err = runRequestInterrupt(os.Args[2:])
	case "ack-interrupt":
		err = runAckInterrupt(os.Args[2:])
	case "clear-interrupt":
		err = runClearInterrupt(os.Args[2:])
	default:
		err = fmt.Errorf("unsupported command: %s", os.Args[1])
	}
	if err != nil {
		fail(err)
	}
}

func runBootstrap(args []string) error {
	flags := flag.NewFlagSet("bootstrap", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	sessionDir := flags.String("session-dir", "", "session dir")
	if err := flags.Parse(args); err != nil {
		return err
	}
	store := eventbus.NewStore(*sessionDir)
	sessionID, err := readSessionID(*sessionDir)
	if err != nil {
		return err
	}
	if err := store.Bootstrap(eventbus.BootstrapOptions{SessionID: sessionID, SessionDir: *sessionDir, MetadataRef: "metadata.md"}); err != nil {
		return err
	}
	return writeJSON(map[string]string{"status": "ok"})
}

func runList(args []string) error {
	flags := flag.NewFlagSet("list", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	sessionDir := flags.String("session-dir", "", "session dir")
	if err := flags.Parse(args); err != nil {
		return err
	}
	store := eventbus.NewStore(*sessionDir)
	events, err := store.List()
	if err != nil {
		return err
	}
	return writeJSON(events)
}

func runAcquireLock(args []string) error {
	flags := flag.NewFlagSet("acquire-lock", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	sessionDir := flags.String("session-dir", "", "session dir")
	holder := flags.String("holder", "", "holder")
	ttlSeconds := flags.Int("ttl-seconds", 30, "ttl seconds")
	if err := flags.Parse(args); err != nil {
		return err
	}
	lease, err := eventbus.NewStore(*sessionDir).AcquireLock(*holder, time.Duration(*ttlSeconds)*time.Second)
	if err != nil {
		return err
	}
	return writeJSON(lease)
}

func runRenewLock(args []string) error {
	flags := flag.NewFlagSet("renew-lock", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	sessionDir := flags.String("session-dir", "", "session dir")
	ttlSeconds := flags.Int("ttl-seconds", 30, "ttl seconds")
	lease, err := parseLeaseFlags(flags, args)
	if err != nil {
		return err
	}
	renewed, err := eventbus.NewStore(*sessionDir).RenewLock(lease, time.Duration(*ttlSeconds)*time.Second)
	if err != nil {
		return err
	}
	return writeJSON(renewed)
}

func runReleaseLock(args []string) error {
	flags := flag.NewFlagSet("release-lock", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	sessionDir := flags.String("session-dir", "", "session dir")
	lease, err := parseLeaseFlags(flags, args)
	if err != nil {
		return err
	}
	if err := eventbus.NewStore(*sessionDir).ReleaseLock(lease); err != nil {
		return err
	}
	return writeJSON(map[string]string{"status": "released"})
}

func runAppend(args []string) error {
	flags := flag.NewFlagSet("append", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	sessionDir := flags.String("session-dir", "", "session dir")
	eventType := flags.String("event-type", "", "event type")
	producer := flags.String("producer", "", "producer")
	roleID := flags.String("role-id", "", "role id")
	correlationID := flags.String("correlation-id", "", "correlation id")
	causationEventID := flags.String("causation-event-id", "", "causation event id")
	payloadRef := flags.String("payload-ref", "", "payload ref")
	summary := flags.String("summary", "", "summary")
	lease, err := parseLeaseFlags(flags, args)
	if err != nil {
		return err
	}
	sessionID, err := readSessionID(*sessionDir)
	if err != nil {
		return err
	}
	event, err := eventbus.NewStore(*sessionDir).Append(lease, eventbus.AppendEvent{
		EventType:        *eventType,
		Producer:         *producer,
		SessionID:        sessionID,
		RoleID:           *roleID,
		CorrelationID:    *correlationID,
		CausationEventID: *causationEventID,
		PayloadRef:       *payloadRef,
		Summary:          *summary,
	})
	if err != nil {
		return err
	}
	return writeJSON(event)
}

func runCommitOffset(args []string) error {
	flags := flag.NewFlagSet("commit-offset", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	sessionDir := flags.String("session-dir", "", "session dir")
	consumerID := flags.String("consumer-id", "", "consumer id")
	lastEventID := flags.String("last-event-id", "", "last event id")
	lastSequence := flags.Int64("last-sequence", 0, "last sequence")
	note := flags.String("note", "", "note")
	lease, err := parseLeaseFlags(flags, args)
	if err != nil {
		return err
	}
	if err := eventbus.NewStore(*sessionDir).CommitOffset(lease, eventbus.OffsetCommit{
		ConsumerID:   *consumerID,
		LastEventID:  *lastEventID,
		LastSequence: *lastSequence,
		Note:         *note,
	}); err != nil {
		return err
	}
	return writeJSON(map[string]string{"status": "ok"})
}

func runRequestInterrupt(args []string) error {
	flags := flag.NewFlagSet("request-interrupt", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	sessionDir := flags.String("session-dir", "", "session dir")
	scope := flags.String("scope", "", "scope")
	targetRoleID := flags.String("target-role-id", "", "target role id")
	source := flags.String("source", "", "source")
	reason := flags.String("reason", "", "reason")
	lease, err := parseLeaseFlags(flags, args)
	if err != nil {
		return err
	}
	record, err := eventbus.NewStore(*sessionDir).RequestInterrupt(lease, eventbus.InterruptRequest{
		Scope:        *scope,
		TargetRoleID: *targetRoleID,
		Source:       *source,
		Reason:       *reason,
	})
	if err != nil {
		return err
	}
	return writeJSON(record)
}

func runAckInterrupt(args []string) error {
	flags := flag.NewFlagSet("ack-interrupt", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	sessionDir := flags.String("session-dir", "", "session dir")
	interruptID := flags.String("interrupt-id", "", "interrupt id")
	lease, err := parseLeaseFlags(flags, args)
	if err != nil {
		return err
	}
	record, err := eventbus.NewStore(*sessionDir).AcknowledgeInterrupt(lease, *interruptID)
	if err != nil {
		return err
	}
	return writeJSON(record)
}

func runClearInterrupt(args []string) error {
	flags := flag.NewFlagSet("clear-interrupt", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	sessionDir := flags.String("session-dir", "", "session dir")
	interruptID := flags.String("interrupt-id", "", "interrupt id")
	lease, err := parseLeaseFlags(flags, args)
	if err != nil {
		return err
	}
	record, err := eventbus.NewStore(*sessionDir).ClearInterrupt(lease, *interruptID)
	if err != nil {
		return err
	}
	return writeJSON(record)
}

func parseLeaseFlags(flags *flag.FlagSet, args []string) (eventbus.Lease, error) {
	holder := flags.String("holder", "", "holder")
	leaseToken := flags.String("lease-token", "", "lease token")
	leaseVersion := flags.Int64("lease-version", 0, "lease version")
	if err := flags.Parse(args); err != nil {
		return eventbus.Lease{}, err
	}
	return eventbus.Lease{
		Holder:       *holder,
		LeaseToken:   *leaseToken,
		LeaseVersion: *leaseVersion,
	}, nil
}

func readSessionID(sessionDir string) (string, error) {
	content, err := os.ReadFile(filepath.Join(sessionDir, "metadata.md"))
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(content), "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "- session_id:") {
			continue
		}
		return strings.TrimSpace(strings.TrimPrefix(trimmed, "- session_id:")), nil
	}
	return "", errors.New("session_id not found in metadata.md")
}

func writeJSON(value any) error {
	return json.NewEncoder(os.Stdout).Encode(value)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err.Error())
	os.Exit(1)
}

func init() {
	flag.CommandLine.SetOutput(os.Stderr)
}
