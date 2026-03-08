package orchestrator

import (
	"path/filepath"
)

func readConclusion(path string) (Conclusion, error) {
	content, err := readFile(path)
	if err != nil {
		return Conclusion{}, err
	}
	values := leadingValues(content)
	conclusion := Conclusion{
		RoleID:     values["role_id"],
		StageID:    values["stage_id"],
		TaskID:     values["task_id"],
		Status:     values["status"],
		Summary:    values["summary"],
		OutputRefs: values["output_refs"],
		UpdatedAt:  values["updated_at"],
	}
	if conclusion.RoleID == "" || conclusion.StageID == "" || conclusion.TaskID == "" {
		return Conclusion{}, ErrInvalidConclusion
	}
	return conclusion, nil
}

func conclusionPath(roleID string) string {
	return filepath.ToSlash(filepath.Join("roles", roleID, "conclusion.md"))
}
