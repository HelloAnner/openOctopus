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

func TestBuildGraphExpandsRepeatTransitionIntoLinearRounds(t *testing.T) {
	graph, err := buildGraph(model.RuntimeConfig{
		Stages: []model.StageConfig{
			{ID: "split_prd", Name: "Split PRD", Role: "prd_splitter"},
			{ID: "review_prd", Name: "Review PRD", Role: "prd_reviewer"},
		},
		Transitions: []model.TransitionConfig{
			{From: "split_prd", To: "review_prd"},
			{From: "review_prd", To: "split_prd", Repeat: model.RepeatConfig{MaxRounds: 3, OnComplete: model.EndStage}},
		},
	})
	if err != nil {
		t.Fatalf("build graph with repeat: %v", err)
	}
	expectedOrder := []string{
		"split_prd__round_01",
		"review_prd__round_01",
		"split_prd__round_02",
		"review_prd__round_02",
		"split_prd__round_03",
		"review_prd__round_03",
	}
	if len(graph.Order) != len(expectedOrder) {
		t.Fatalf("expected %d expanded stages, got %d: %+v", len(expectedOrder), len(graph.Order), graph.Order)
	}
	for index, item := range expectedOrder {
		if graph.Order[index] != item {
			t.Fatalf("expected order[%d]=%q, got %q", index, item, graph.Order[index])
		}
	}
	if len(graph.EntryStageIDs) != 1 || graph.EntryStageIDs[0] != "split_prd__round_01" {
		t.Fatalf("unexpected entry stages: %+v", graph.EntryStageIDs)
	}
	if graph.Stages["review_prd__round_02"].NextStageID != "split_prd__round_03" {
		t.Fatalf("expected second review to lead to round 3 split, got %+v", graph.Stages["review_prd__round_02"])
	}
	if graph.Stages["review_prd__round_03"].NextStageID != model.EndStage {
		t.Fatalf("expected final review to end workflow, got %+v", graph.Stages["review_prd__round_03"])
	}
}
