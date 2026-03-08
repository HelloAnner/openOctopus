/*
Package humangate 提供人工打断、人工补充与恢复续跑的服务入口。
Author: Anner
Created on 2026/3/8
*/
package humangate

import "github.com/anner/openoctopus/internal/eventbus"

const (
	humanGateHolder = "human-gate"
	humanGateSource = "human-gate"
	injectKind      = "inject"
)

type filePaths struct {
	humanMessages string
	schedule      string
	blockers      string
	decisionLog   string
	sessionState  string
}

// Service 提供 human-gate 的文件协议与 event-bus 组合能力。
type Service struct {
	sessionDir string
	paths      filePaths
	bus        *eventbus.Store
}

// InjectOptions 描述一次人工补充输入。
type InjectOptions struct {
	RoleID  string
	Message string
}

// InjectResult 返回写入后的消息结果。
type InjectResult struct {
	MessageID string
	Path      string
}

// InterruptOptions 描述单角色中断请求。
type InterruptOptions struct {
	RoleID string
	Reason string
}

// InterruptAllOptions 描述批量中断请求。
type InterruptAllOptions struct {
	Reason string
}

// InterruptAllResult 返回本次批量中断的结果。
type InterruptAllResult struct {
	RequestedCount int
}

// ResumeOptions 描述恢复执行请求。
type ResumeOptions struct {
	RoleID string
}

// ResumeResult 返回恢复动作的统计结果。
type ResumeResult struct {
	ClearedInterrupts int
	RequeuedStages    int
}

type scheduleState struct {
	ScheduleVersion     int
	WorkflowStatus      string
	ActiveDispatchCount int
	UpdatedAt           string
	Stages              []stageEntry
}

type stageEntry struct {
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
