package orchestrator

import (
	"fmt"
	"strings"

	"github.com/anner/openoctopus/internal/config/model"
)

const repeatedStageSeparator = "__round_"

func hasRepeat(config model.RepeatConfig) bool {
	return config.MaxRounds > 0 || strings.TrimSpace(config.OnComplete) != ""
}

func expandedStageID(stageID string, round int) string {
	return fmt.Sprintf("%s%s%02d", stageID, repeatedStageSeparator, round)
}

func expandedStageName(stageName string, round int) string {
	return fmt.Sprintf("%s [Round %02d]", stageName, round)
}

func splitExpandedStageID(stageID string) (string, int, bool) {
	index := strings.LastIndex(stageID, repeatedStageSeparator)
	if index <= 0 {
		return "", 0, false
	}
	baseID := stageID[:index]
	round := atoi(stageID[index+len(repeatedStageSeparator):])
	if round <= 0 {
		return "", 0, false
	}
	return baseID, round, true
}
