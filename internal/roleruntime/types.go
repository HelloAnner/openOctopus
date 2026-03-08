package roleruntime

import (
	"path/filepath"

	"github.com/anner/openoctopus/internal/config/model"
	"github.com/anner/openoctopus/internal/eventbus"
)

const (
	statusIdle        = "IDLE"
	statusRunning     = "RUNNING"
	statusCompleted   = "COMPLETED"
	statusBlocked     = "BLOCKED"
	statusFailed      = "FAILED"
	statusInterrupted = "INTERRUPTED"
	statusResetPending = "RESET_PENDING"

	resetStatusIdle      = "IDLE"
	resetStatusRequested = "REQUESTED"
	resetStatusApplied   = "APPLIED"
)

type Engine struct {
	sessionDir string
	paths      filePaths
	bus        *eventbus.Store
}

type filePaths struct {
	effectiveConfig string
	requirement     string
	rolesDir        string
}

type RoleTickResult struct {
	RoleID      string
	Progressed  bool
	TurnSeq     int
	Status      string
	Skipped     bool
}

type TickAllResult struct {
	Progressed    bool
	ExecutedRoles int
	TurnCount     int
}

type roleState struct {
	RoleID             string
	SessionGeneration  int
	Status             string
	CurrentStageID     string
	CurrentTaskID      string
	CurrentTurnSeq     int
	ContextVersion     int
	InboxVersion       int
	LastConsumedEventID string
	LastConclusionStatus string
	ExecutorProvider   string
	UpdatedAt          string
}

type roleReset struct {
	SessionGeneration int
	Status            string
	RequestedBy       string
	Reason            string
	RequestedAt       string
	AppliedAt         string
	LastClearedTaskID string
}

type roleHeartbeat struct {
	HeartbeatVersion int
	RoleID           string
	Status           string
	CurrentTaskID    string
	CurrentTurnSeq   int
	LastSeenAt       string
	ExpireAt         string
	SessionGeneration int
	UpdatedAt        string
}

type roleOutbox struct {
	OutboxVersion  int
	RoleID         string
	StageID        string
	TaskID         string
	TurnSeq        int
	Status         string
	ConclusionRef  string
	TurnOutputRef  string
	UpdatedAt      string
}

type roleContext struct {
	ContextVersion int
	TaskID         string
	StageID        string
	RoleID         string
	Attempt        int
	SystemPrompt   string
}

type roleInbox struct {
	InboxVersion    int
	TaskID          string
	StageID         string
	RoleID          string
	Status          string
	DispatchEventID string
	ContextVersion  int
}

type ExecuteRequest struct {
	SessionDir         string
	Role               model.RoleConfig
	Profile            model.LLMProfile
	Context            roleContext
	Inbox              roleInbox
	TurnSeq            int
	Prompt             string
}

type ExecuteResult struct {
	Provider   string
	Command    string
	ExitCode   int
	DurationMS int64
	Stdout     string
	Stderr     string
	RawOutput  string
}

type roleResult struct {
	Status    string
	Summary   string
	OutputRefs string
}

type executor interface {
	Execute(request ExecuteRequest) (ExecuteResult, error)
}

func NewEngine(sessionDir string) *Engine {
	return &Engine{
		sessionDir: sessionDir,
		paths: filePaths{
			effectiveConfig: filepath.Join(sessionDir, "state", "effective_config.yaml"),
			requirement:     filepath.Join(sessionDir, "planner", "requirement.snapshot.md"),
			rolesDir:        filepath.Join(sessionDir, "roles"),
		},
		bus: eventbus.NewStore(sessionDir),
	}
}
