package orchestrator

import (
	"path/filepath"
	"sort"
	"time"

	"github.com/anner/openoctopus/internal/config/model"
	"github.com/anner/openoctopus/internal/eventbus"
)

const (
	stageStatusPending      = "PENDING"
	stageStatusReady        = "READY"
	stageStatusDispatched   = "DISPATCHED"
	stageStatusCompleted    = "COMPLETED"
	stageStatusRetryPending = "RETRY_PENDING"
	stageStatusBlocked      = "BLOCKED"
	stageStatusFailed       = "FAILED"

	workflowStatusReady        = "READY"
	workflowStatusRunning      = "RUNNING"
	workflowStatusWaitingHuman = "WAITING_HUMAN"
	workflowStatusCompleted    = "COMPLETED"
	workflowStatusFailed       = "FAILED"

	conclusionSuccess    = "SUCCESS"
	conclusionNeedsRetry = "NEEDS_RETRY"
	conclusionBlocked    = "BLOCKED"
	conclusionFailed     = "FAILED"

	orchestratorConsumerID = "orchestrator/master"
	orchestratorHolder     = "orchestrator/master"
)

var nowFunc = time.Now

type Engine struct {
	sessionDir string
	paths      filePaths
	bus        *eventbus.Store
}

type filePaths struct {
	effectiveConfig string
	metadata        string
	sessionState    string
	humanMessages   string
	requirement     string
	schedule        string
	taskBoard       string
	taskGraph       string
	globalProgress  string
	blockers        string
	dispatchLog     string
	decisionLog     string
	rolesDir        string
}

type Graph struct {
	Stages        map[string]StageNode
	Order         []string
	EntryStageIDs []string
}

type StageNode struct {
	Config      model.StageConfig
	NextStageID string
}

type HumanMessage struct {
	MessageID string
	Source    string
	CreatedAt string
	Content   string
}

type RequirementSnapshot struct {
	SnapshotVersion    int
	WorkflowID         string
	WorkflowName       string
	HumanMessageCursor string
	SourceMessageCount int
	PlannerStatus      string
	UpdatedAt          string
	WorkflowSummary    string
	LatestMessages     string
	DispatchBrief      string
}

type Schedule struct {
	ScheduleVersion     int
	WorkflowStatus      string
	ActiveDispatchCount int
	UpdatedAt           string
	Stages              []StageSchedule
}

type StageSchedule struct {
	StageID           string
	StageName         string
	RoleID            string
	Status            string
	Attempt           int
	LastTaskID        string
	LastConclusionRef string
	NextStageID       string
	UpdatedAt         string
}

type DispatchTask struct {
	TaskID         string
	StageID        string
	RoleID         string
	Attempt        int
	ContextVersion int
	InboxVersion   int
}

type TickResult struct {
	WorkflowStatus  string
	DispatchedCount int
	SnapshotUpdated bool
}

type Conclusion struct {
	RoleID     string
	StageID    string
	TaskID     string
	Status     string
	Summary    string
	OutputRefs string
	UpdatedAt  string
}

type sessionState struct {
	SessionID          string
	Status             string
	CurrentStageID     string
	CurrentRoleID      string
	CheckpointSeq      string
	LastEvent          string
	CreatedAt          string
	UpdatedAt          string
	HumanMessageCursor string
}

func newFilePaths(sessionDir string) filePaths {
	plannerDir := filepath.Join(sessionDir, "planner")
	return filePaths{
		effectiveConfig: filepath.Join(sessionDir, "state", "effective_config.yaml"),
		metadata:        filepath.Join(sessionDir, "metadata.md"),
		sessionState:    filepath.Join(sessionDir, "session.state.md"),
		humanMessages:   filepath.Join(plannerDir, "human_messages.md"),
		requirement:     filepath.Join(plannerDir, "requirement.snapshot.md"),
		schedule:        filepath.Join(plannerDir, "master_schedule.md"),
		taskBoard:       filepath.Join(plannerDir, "task_board.md"),
		taskGraph:       filepath.Join(plannerDir, "task_graph.mmd"),
		globalProgress:  filepath.Join(plannerDir, "global_progress.md"),
		blockers:        filepath.Join(plannerDir, "blockers.md"),
		dispatchLog:     filepath.Join(plannerDir, "dispatch_log.md"),
		decisionLog:     filepath.Join(plannerDir, "decision_log.md"),
		rolesDir:        filepath.Join(sessionDir, "roles"),
	}
}

func NewEngine(sessionDir string) *Engine {
	return &Engine{sessionDir: sessionDir, paths: newFilePaths(sessionDir), bus: eventbus.NewStore(sessionDir)}
}

func sortStageSchedules(items []StageSchedule) {
	sort.Slice(items, func(left int, right int) bool {
		return items[left].StageID < items[right].StageID
	})
}
