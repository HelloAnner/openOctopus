package orchestrator

import (
	"errors"
	"path/filepath"
	"strings"

	artifactstore "github.com/anner/openoctopus/internal/artifact"
	"github.com/anner/openoctopus/internal/config/model"
	"github.com/anner/openoctopus/internal/eventbus"
)

type artifactInputBinding struct {
	Ref         string
	Version     int
	ContentRef  string
	ManifestRef string
}

type artifactOutputBinding struct {
	Name         string
	SuggestedRef string
}

type roleOutboxRecord struct {
	TurnOutputRef string
}

func findStageConfig(config model.RuntimeConfig, stageID string) (model.StageConfig, bool) {
	for _, stage := range config.Stages {
		if stage.ID == stageID {
			return stage, true
		}
	}
	baseStageID, round, ok := splitExpandedStageID(stageID)
	if !ok {
		return model.StageConfig{}, false
	}
	for _, stage := range config.Stages {
		if stage.ID != baseStageID {
			continue
		}
		stage.ID = stageID
		stage.Name = expandedStageName(stage.Name, round)
		return stage, true
	}
	return model.StageConfig{}, false
}

func (e *Engine) publishArtifacts(stageConfig model.StageConfig, stage StageSchedule, conclusion Conclusion, lease eventbus.Lease) error {
	outputs := buildArtifactOutputs(stageConfig)
	if len(outputs) == 0 {
		return nil
	}
	outbox, _ := readOutbox(filepath.Join(e.paths.rolesDir, stage.RoleID, "outbox.md"))
	explicitRefs := splitRefList(conclusion.OutputRefs)
	store := artifactstore.NewStore(e.sessionDir)
	for index, output := range outputs {
		candidates := make([]string, 0, 3)
		if index < len(explicitRefs) {
			candidates = append(candidates, explicitRefs[index])
		}
		candidates = append(candidates, output.SuggestedRef, outbox.TurnOutputRef)
		published, err := publishWithFallback(store, output.Name, stage, candidates)
		if err != nil {
			return err
		}
		_, err = e.bus.Append(lease, eventbus.AppendEvent{EventType: "ARTIFACT_PUBLISHED", Producer: "artifact", SessionID: readStateSessionIDOrEmpty(e.paths.sessionState), RoleID: stage.RoleID, PayloadRef: published.ManifestRef, Summary: output.Name + " published"})
		if err != nil {
			return err
		}
	}
	return nil
}

func publishWithFallback(store *artifactstore.Store, artifactName string, stage StageSchedule, candidates []string) (artifactstore.ArtifactVersion, error) {
	var lastErr error
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		published, err := store.Publish(artifactstore.PublishRequest{ArtifactName: artifactName, StageID: stage.StageID, RoleID: stage.RoleID, TaskID: stage.LastTaskID, SourceRef: candidate, ConclusionRef: conclusionPath(stage.RoleID), TurnOutputRef: filepath.ToSlash(filepath.Join("roles", stage.RoleID, "turns", "0001-output.md"))})
		if err == nil {
			return published, nil
		}
		if !errors.Is(err, artifactstore.ErrSourceNotFound) {
			return artifactstore.ArtifactVersion{}, err
		}
		lastErr = err
	}
	if lastErr != nil {
		return artifactstore.ArtifactVersion{}, lastErr
	}
	return artifactstore.ArtifactVersion{}, artifactstore.ErrSourceNotFound
}

func readOutbox(path string) (roleOutboxRecord, error) {
	content, err := readFile(path)
	if err != nil {
		return roleOutboxRecord{}, err
	}
	values := leadingValues(content)
	return roleOutboxRecord{TurnOutputRef: values["turn_output_ref"]}, nil
}

func splitRefList(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	results := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			results = append(results, trimmed)
		}
	}
	return results
}
