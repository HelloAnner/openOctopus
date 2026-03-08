package eventbus

import (
	"fmt"
	"path/filepath"
	"time"
)

const (
	LockStatusFree    = "FREE"
	LockStatusHeld    = "HELD"
	LockStatusExpired = "EXPIRED"

	InterruptScopeRole = "role"

	InterruptStatusRequested    = "REQUESTED"
	InterruptStatusAcknowledged = "ACKNOWLEDGED"
	InterruptStatusCleared      = "CLEARED"
)

var nowFunc = time.Now

var leaseTokenFunc = func() string {
	return fmt.Sprintf("lease-%d", time.Now().UnixNano())
}

type Store struct {
	sessionDir string
	paths      busPaths
}

type busPaths struct {
	events     string
	lock       string
	offsets    string
	interrupts string
	metadata   string
}

type BootstrapOptions struct {
	SessionID   string `json:"session_id"`
	SessionDir  string `json:"session_dir"`
	WorkflowID  string `json:"workflow_id"`
	MetadataRef string `json:"metadata_ref"`
}

type AppendEvent struct {
	EventType        string `json:"event_type"`
	Producer         string `json:"producer"`
	SessionID        string `json:"session_id"`
	RoleID           string `json:"role_id"`
	CorrelationID    string `json:"correlation_id"`
	CausationEventID string `json:"causation_event_id"`
	PayloadRef       string `json:"payload_ref"`
	Summary          string `json:"summary"`
}

type Event struct {
	EventID          string `json:"event_id"`
	Sequence         int64  `json:"sequence"`
	Timestamp        string `json:"ts"`
	EventType        string `json:"event_type"`
	Producer         string `json:"producer"`
	SessionID        string `json:"session_id"`
	RoleID           string `json:"role_id"`
	CorrelationID    string `json:"correlation_id"`
	CausationEventID string `json:"causation_event_id"`
	PayloadRef       string `json:"payload_ref"`
	Summary          string `json:"summary"`
	PrevEventHash    string `json:"prev_event_hash"`
	EventHash        string `json:"event_hash"`
}

type Lease struct {
	Status        string `json:"status"`
	Holder        string `json:"holder"`
	LeaseToken    string `json:"lease_token"`
	LeaseVersion  int64  `json:"lease_version"`
	AcquiredAt    string `json:"acquired_at"`
	RenewedAt     string `json:"renewed_at"`
	ExpireAt      string `json:"expire_at"`
	LastOperation string `json:"last_operation"`
}

type OffsetCommit struct {
	ConsumerID   string `json:"consumer_id"`
	LastEventID  string `json:"last_event_id"`
	LastSequence int64  `json:"last_sequence"`
	Note         string `json:"note"`
}

type OffsetEntry struct {
	ConsumerID   string `json:"consumer_id"`
	LastEventID  string `json:"last_event_id"`
	LastSequence int64  `json:"last_sequence"`
	UpdatedAt    string `json:"updated_at"`
	Note         string `json:"note"`
}

type InterruptRequest struct {
	Scope        string `json:"scope"`
	TargetRoleID string `json:"target_role_id"`
	Source       string `json:"source"`
	Reason       string `json:"reason"`
}

type InterruptRecord struct {
	InterruptID    string `json:"interrupt_id"`
	Scope          string `json:"scope"`
	TargetRoleID   string `json:"target_role_id"`
	Source         string `json:"source"`
	Reason         string `json:"reason"`
	Status         string `json:"status"`
	RequestEventID string `json:"request_event_id"`
	AckEventID     string `json:"ack_event_id"`
	ClearEventID   string `json:"clear_event_id"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

func NewStore(sessionDir string) *Store {
	return &Store{
		sessionDir: sessionDir,
		paths: busPaths{
			events:     filepath.Join(sessionDir, "bus", "events.md"),
			lock:       filepath.Join(sessionDir, "bus", "lock.md"),
			offsets:    filepath.Join(sessionDir, "bus", "offsets.md"),
			interrupts: filepath.Join(sessionDir, "bus", "interrupts.md"),
			metadata:   filepath.Join(sessionDir, "metadata.md"),
		},
	}
}
