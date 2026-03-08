package orchestrator

import (
	"sort"

	"github.com/anner/openoctopus/internal/config/model"
)

func buildGraph(config model.RuntimeConfig) (Graph, error) {
	transitions, repeated, predecessors, err := indexTransitions(config.Transitions)
	if err != nil {
		return Graph{}, err
	}
	if repeated == nil {
		return buildDirectGraph(config.Stages, transitions, predecessors), nil
	}
	return buildRepeatedGraph(config.Stages, transitions, predecessors, *repeated)
}

func indexTransitions(items []model.TransitionConfig) (map[string]model.TransitionConfig, *model.TransitionConfig, map[string]int, error) {
	indexed := make(map[string]model.TransitionConfig, len(items))
	predecessors := make(map[string]int)
	seenFrom := make(map[string]struct{}, len(items))
	var repeated *model.TransitionConfig
	for index := range items {
		item := items[index]
		if !supportsTransitionShape(item) {
			return nil, nil, nil, ErrUnsupportedTransitionShape
		}
		if _, exists := seenFrom[item.From]; exists {
			return nil, nil, nil, ErrUnsupportedTransitionShape
		}
		seenFrom[item.From] = struct{}{}
		indexed[item.From] = item
		if hasRepeat(item.Repeat) {
			if repeated != nil {
				return nil, nil, nil, ErrUnsupportedTransitionShape
			}
			copied := item
			repeated = &copied
			continue
		}
		if item.To != model.EndStage {
			predecessors[item.To]++
		}
	}
	return indexed, repeated, predecessors, nil
}

func supportsTransitionShape(item model.TransitionConfig) bool {
	if item.To == "" {
		return false
	}
	if item.Condition.Type != "" || item.Condition.Expr != "" || item.Condition.Mode != "" || len(item.Condition.Rules) != 0 {
		return false
	}
	if item.OnTrue != "" || item.OnFalse != "" {
		return false
	}
	return true
}

func buildDirectGraph(stages []model.StageConfig, transitions map[string]model.TransitionConfig, predecessors map[string]int) Graph {
	nodes := make(map[string]StageNode, len(stages))
	order := make([]string, 0, len(stages))
	for _, item := range stages {
		nextStageID := model.EndStage
		if transition, ok := transitions[item.ID]; ok {
			nextStageID = transition.To
		}
		nodes[item.ID] = StageNode{Config: item, NextStageID: nextStageID}
		order = append(order, item.ID)
	}
	return Graph{Stages: nodes, Order: order, EntryStageIDs: entryStageIDs(order, predecessors)}
}

func buildRepeatedGraph(stages []model.StageConfig, transitions map[string]model.TransitionConfig, predecessors map[string]int, repeated model.TransitionConfig) (Graph, error) {
	baseConfigs := stageConfigMap(stages)
	entries := entryStageIDs(stageOrder(stages), predecessors)
	if len(entries) != 1 {
		return Graph{}, ErrUnsupportedTransitionShape
	}
	prefix, err := collectPrefix(entries[0], repeated.To, transitions)
	if err != nil {
		return Graph{}, err
	}
	loopStages, err := collectLoopStages(repeated.To, repeated.From, transitions)
	if err != nil {
		return Graph{}, err
	}
	suffix, err := collectSuffix(repeated.Repeat.OnComplete, transitions)
	if err != nil {
		return Graph{}, err
	}
	if err := validateRepeatedCoverage(stages, prefix, loopStages, suffix); err != nil {
		return Graph{}, err
	}
	return expandRepeatedGraph(baseConfigs, prefix, loopStages, suffix, repeated.Repeat.MaxRounds), nil
}

func stageConfigMap(stages []model.StageConfig) map[string]model.StageConfig {
	values := make(map[string]model.StageConfig, len(stages))
	for _, item := range stages {
		values[item.ID] = item
	}
	return values
}

func stageOrder(stages []model.StageConfig) []string {
	order := make([]string, 0, len(stages))
	for _, item := range stages {
		order = append(order, item.ID)
	}
	return order
}

func entryStageIDs(order []string, predecessors map[string]int) []string {
	entries := make([]string, 0)
	for _, stageID := range order {
		if predecessors[stageID] == 0 {
			entries = append(entries, stageID)
		}
	}
	sort.Strings(entries)
	return entries
}

func collectPrefix(entry string, target string, transitions map[string]model.TransitionConfig) ([]string, error) {
	if entry == target {
		return nil, nil
	}
	values := make([]string, 0)
	current := entry
	visited := make(map[string]struct{})
	for current != target {
		if _, exists := visited[current]; exists {
			return nil, ErrUnsupportedTransitionShape
		}
		visited[current] = struct{}{}
		values = append(values, current)
		next, ok := nextStageID(transitions, current)
		if !ok {
			return nil, ErrUnsupportedTransitionShape
		}
		current = next
	}
	return values, nil
}

func collectLoopStages(start string, end string, transitions map[string]model.TransitionConfig) ([]string, error) {
	values := make([]string, 0)
	current := start
	visited := make(map[string]struct{})
	for {
		if _, exists := visited[current]; exists {
			return nil, ErrUnsupportedTransitionShape
		}
		visited[current] = struct{}{}
		values = append(values, current)
		if current == end {
			return values, nil
		}
		next, ok := nextStageID(transitions, current)
		if !ok {
			return nil, ErrUnsupportedTransitionShape
		}
		current = next
	}
}

func collectSuffix(start string, transitions map[string]model.TransitionConfig) ([]string, error) {
	if start == "" || start == model.EndStage {
		return nil, nil
	}
	values := make([]string, 0)
	current := start
	visited := make(map[string]struct{})
	for current != model.EndStage {
		if _, exists := visited[current]; exists {
			return nil, ErrUnsupportedTransitionShape
		}
		visited[current] = struct{}{}
		values = append(values, current)
		next, ok := nextStageID(transitions, current)
		if !ok {
			current = model.EndStage
			continue
		}
		current = next
	}
	return values, nil
}

func validateRepeatedCoverage(stages []model.StageConfig, prefix []string, loop []string, suffix []string) error {
	seen := make(map[string]struct{}, len(stages))
	for _, group := range [][]string{prefix, loop, suffix} {
		for _, stageID := range group {
			if _, exists := seen[stageID]; exists {
				return ErrUnsupportedTransitionShape
			}
			seen[stageID] = struct{}{}
		}
	}
	if len(seen) != len(stages) {
		return ErrUnsupportedTransitionShape
	}
	return nil
}

func expandRepeatedGraph(baseConfigs map[string]model.StageConfig, prefix []string, loop []string, suffix []string, maxRounds int) Graph {
	expanded := make([]model.StageConfig, 0, len(prefix)+len(suffix)+(len(loop)*maxRounds))
	for _, stageID := range prefix {
		expanded = append(expanded, baseConfigs[stageID])
	}
	for round := 1; round <= maxRounds; round++ {
		for _, stageID := range loop {
			stage := baseConfigs[stageID]
			stage.ID = expandedStageID(stageID, round)
			stage.Name = expandedStageName(stage.Name, round)
			expanded = append(expanded, stage)
		}
	}
	for _, stageID := range suffix {
		expanded = append(expanded, baseConfigs[stageID])
	}
	nodes := make(map[string]StageNode, len(expanded))
	order := make([]string, 0, len(expanded))
	for index, item := range expanded {
		nextStageID := model.EndStage
		if index+1 < len(expanded) {
			nextStageID = expanded[index+1].ID
		}
		nodes[item.ID] = StageNode{Config: item, NextStageID: nextStageID}
		order = append(order, item.ID)
	}
	return Graph{Stages: nodes, Order: order, EntryStageIDs: []string{expanded[0].ID}}
}

func nextStageID(transitions map[string]model.TransitionConfig, stageID string) (string, bool) {
	transition, ok := transitions[stageID]
	if !ok {
		return model.EndStage, false
	}
	return transition.To, true
}
