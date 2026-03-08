/*
Package recovery types 定义 recovery 首版的输入输出模型。
Author: Anner
Created on 2026/3/8
*/
package recovery

import (
	"path/filepath"

	"github.com/anner/openoctopus/internal/eventbus"
)

const (
	workflowStatusReady        = "READY"
	workflowStatusRunning      = "RUNNING"
	workflowStatusWaitingHuman = "WAITING_HUMAN"
	workflowStatusCompleted    = "COMPLETED"
	workflowStatusFailed       = "FAILED"
)

type filePaths struct {
	metadata        string
	sessionState    string
	effectiveConfig string
	events          string
	interrupts      string
	schedule        string
	blockers        string
	rolesDir        string
	checkpointsDir  string
	replay          string
}

// Service 提供 recovery 的校验、修复与恢复入口。
type Service struct {
	sessionDir string
	paths      filePaths
	bus        *eventbus.Store
}

// RecoverOptions 描述一次恢复动作。
type RecoverOptions struct{}

// RecoverResult 描述恢复的最小结果。
type RecoverResult struct {
	SessionID       string   `json:"session_id"`
	SessionDir      string   `json:"session_dir"`
	RecoveredStatus string   `json:"recovered_status"`
	Continued       bool     `json:"continued"`
	RepairedFiles   []string `json:"repaired_files"`
	CheckpointRef   string   `json:"checkpoint_ref"`
	ReplayRef       string   `json:"replay_ref"`
	Reason          string   `json:"reason,omitempty"`
	CanContinue     bool     `json:"-"`
	CheckedFiles    []string `json:"-"`
}

// CheckpointInput 描述一次 checkpoint 落盘请求。
type CheckpointInput struct {
	Kind   string
	Source string
}

// CheckpointRecord 返回写入后的 checkpoint 结果。
type CheckpointRecord struct {
	Sequence int
	Path     string
	Ref      string
	Kind     string
}

type sessionValidation struct {
	SessionID       string
	RepairableFiles []string
	CheckedFiles    []string
}

type metadataDoc struct {
	SessionID string
	CreatedAt string
}

type sessionStateDoc struct {
	SessionID          string
	Status             string
	CurrentStageID     string
	CurrentRoleID      string
	CheckpointSeq      int
	LastEvent          string
	CreatedAt          string
	UpdatedAt          string
	HumanMessageCursor string
}

type scheduleDoc struct {
	ScheduleVersion int
	WorkflowStatus  string
	Stages          []stageDoc
}

type stageDoc struct {
	StageID           string
	RoleID            string
	Status            string
	LastConclusionRef string
}

type recoveryView struct {
	SessionID          string
	WorkflowStatus     string
	CurrentStageID     string
	CurrentRoleID      string
	BlockerSummary     string
	LastEvent          string
	CreatedAt          string
	HumanMessageCursor string
	ScheduleVersion    int
	Reason             string
}

// NewService 创建 recovery 服务。
func NewService(sessionDir string) *Service {
	return &Service{
		sessionDir: sessionDir,
		paths: filePaths{
			metadata:        filepath.Join(sessionDir, "metadata.md"),
			sessionState:    filepath.Join(sessionDir, "session.state.md"),
			effectiveConfig: filepath.Join(sessionDir, "state", "effective_config.yaml"),
			events:          filepath.Join(sessionDir, "bus", "events.md"),
			interrupts:      filepath.Join(sessionDir, "bus", "interrupts.md"),
			schedule:        filepath.Join(sessionDir, "planner", "master_schedule.md"),
			blockers:        filepath.Join(sessionDir, "planner", "blockers.md"),
			rolesDir:        filepath.Join(sessionDir, "roles"),
			checkpointsDir:  filepath.Join(sessionDir, "state", "checkpoints"),
			replay:          filepath.Join(sessionDir, "audit", "replay.md"),
		},
		bus: eventbus.NewStore(sessionDir),
	}
}
