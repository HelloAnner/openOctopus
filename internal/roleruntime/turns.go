package roleruntime

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/anner/openoctopus/internal/config/model"
)

func readRoleContext(root string, roleID string) (roleContext, error) {
	content, err := readFile(contextPath(root, roleID))
	if err != nil {
		return roleContext{}, err
	}
	values := leadingValues(content)
	return roleContext{
		ContextVersion: atoi(values["context_version"]),
		TaskID:         values["task_id"],
		StageID:        values["stage_id"],
		RoleID:         values["role_id"],
		Attempt:        atoi(values["attempt"]),
		SystemPrompt:   sectionText(content, "## system_prompt"),
	}, nil
}

func readRoleInbox(root string, roleID string) (roleInbox, error) {
	content, err := readFile(inboxPath(root, roleID))
	if err != nil {
		return roleInbox{}, err
	}
	values := leadingValues(content)
	return roleInbox{
		InboxVersion:    atoi(values["inbox_version"]),
		TaskID:          values["task_id"],
		StageID:         values["stage_id"],
		RoleID:          values["role_id"],
		Status:          values["status"],
		DispatchEventID: values["dispatch_event_id"],
		ContextVersion:  atoi(values["context_version"]),
	}, nil
}

func writeTurnInput(root string, roleID string, request ExecuteRequest, config model.RuntimeConfig) (string, error) {
	readOrder := config.Runtime.RoleRuntime.BootstrapReadOrder
	if len(readOrder) == 0 {
		readOrder = []string{"planner/requirement.snapshot.md", fmt.Sprintf("roles/%s/context.md", roleID), fmt.Sprintf("roles/%s/inbox.md", roleID)}
	}
	mustRead := strings.Join(request.Role.Constraints.MustReadFiles, ",")
	forbidden := strings.Join(request.Role.Constraints.ForbiddenWrites, ",")
	lines := []string{
		"# Role Turn Input",
		"",
		fmt.Sprintf("- turn_seq: %d", request.TurnSeq),
		fmt.Sprintf("- role_id: %s", roleID),
		fmt.Sprintf("- stage_id: %s", request.Inbox.StageID),
		fmt.Sprintf("- task_id: %s", request.Inbox.TaskID),
		fmt.Sprintf("- session_generation: %d", request.Context.Attempt),
		fmt.Sprintf("- context_version: %d", request.Context.ContextVersion),
		fmt.Sprintf("- inbox_version: %d", request.Inbox.InboxVersion),
		fmt.Sprintf("- dispatch_event_id: %s", request.Inbox.DispatchEventID),
		fmt.Sprintf("- read_order: %s", strings.Join(readOrder, ",")),
		fmt.Sprintf("- must_read_files: %s", mustRead),
		fmt.Sprintf("- forbidden_writes: %s", forbidden),
		"",
		"## system_prompt",
		request.Context.SystemPrompt,
		"",
		"## prompt",
		request.Prompt,
	}
	path := filepath.Join(turnsDirPath(root, roleID), turnFileName(request.TurnSeq, "input"))
	return path, atomicWrite(path, []byte(strings.Join(lines, "\n")+"\n"))
}

func writeTurnOutput(root string, roleID string, turnSeq int, result ExecuteResult) (string, error) {
	lines := []string{
		"# Role Turn Output",
		"",
		fmt.Sprintf("- turn_seq: %d", turnSeq),
		fmt.Sprintf("- executor_provider: %s", result.Provider),
		fmt.Sprintf("- command: %s", result.Command),
		fmt.Sprintf("- exit_code: %d", result.ExitCode),
		fmt.Sprintf("- duration_ms: %d", result.DurationMS),
		"",
		"## stdout",
		result.Stdout,
		"",
		"## stderr",
		result.Stderr,
		"",
		result.RawOutput,
	}
	path := filepath.Join(turnsDirPath(root, roleID), turnFileName(turnSeq, "output"))
	return path, atomicWrite(path, []byte(strings.Join(lines, "\n")+"\n"))
}
