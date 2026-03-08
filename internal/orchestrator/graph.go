package orchestrator

import (
	"sort"

	"github.com/anner/openoctopus/internal/config/model"
)

func buildGraph(config model.RuntimeConfig) (Graph, error) {
	stages := make(map[string]StageNode, len(config.Stages))
	order := make([]string, 0, len(config.Stages))
	predecessors := make(map[string]int)
	for _, item := range config.Stages {
		stages[item.ID] = StageNode{Config: item, NextStageID: model.EndStage}
		order = append(order, item.ID)
	}
	seenFrom := make(map[string]struct{})
	for _, item := range config.Transitions {
		if item.To == "" || item.Condition.Type != "" || item.Condition.Expr != "" || item.Condition.Mode != "" || len(item.Condition.Rules) != 0 || item.OnTrue != "" || item.OnFalse != "" {
			return Graph{}, ErrUnsupportedTransitionShape
		}
		if _, exists := seenFrom[item.From]; exists {
			return Graph{}, ErrUnsupportedTransitionShape
		}
		seenFrom[item.From] = struct{}{}
		node := stages[item.From]
		node.NextStageID = item.To
		stages[item.From] = node
		if item.To != model.EndStage {
			predecessors[item.To]++
		}
	}
	entryStageIDs := make([]string, 0)
	for _, stageID := range order {
		if predecessors[stageID] == 0 {
			entryStageIDs = append(entryStageIDs, stageID)
		}
	}
	sort.Strings(entryStageIDs)
	return Graph{Stages: stages, Order: order, EntryStageIDs: entryStageIDs}, nil
}
