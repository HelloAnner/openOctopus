package orchestrator

import (
	"testing"

	"github.com/anner/openoctopus/internal/config/model"
)

func TestBuildGraphSupportsDirectTransitionsAndRejectsUnsupportedShapes(t *testing.T) {
	graph, err := buildGraph(model.RuntimeConfig{
		Stages:      []model.StageConfig{{ID: "stage_a", Name: "Stage A", Role: "agent_a"}},
		Transitions: []model.TransitionConfig{{From: "stage_a", To: model.EndStage}},
	})
	if err != nil {
		t.Fatalf("build graph: %v", err)
	}
	if len(graph.EntryStageIDs) != 1 || graph.EntryStageIDs[0] != "stage_a" {
		t.Fatalf("unexpected entry stages: %+v", graph.EntryStageIDs)
	}
	if graph.Stages["stage_a"].NextStageID != model.EndStage {
		t.Fatalf("unexpected next stage: %+v", graph.Stages["stage_a"])
	}

	_, err = buildGraph(model.RuntimeConfig{
		Stages:      []model.StageConfig{{ID: "stage_a", Name: "Stage A", Role: "agent_a"}},
		Transitions: []model.TransitionConfig{{From: "stage_a", OnTrue: "stage_b"}},
	})
	if err == nil {
		t.Fatal("expected unsupported transition shape error")
	}
}
