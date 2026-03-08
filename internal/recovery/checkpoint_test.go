/*
Package recovery checkpoint_test 验证 recovery checkpoint 追加写能力。
Author: Anner
Created on 2026/3/8
*/
package recovery

import (
	"path/filepath"
	"testing"
)

func TestWriteCheckpointAppendsSequence(t *testing.T) {
	sessionDir := prepareRecoverySession(t)

	first, err := RecordCheckpoint(sessionDir, CheckpointInput{Kind: "stage-stage_a-dispatched", Source: "orchestrator"})
	if err != nil {
		t.Fatalf("write first checkpoint: %v", err)
	}
	if first.Sequence != 1 {
		t.Fatalf("expected first sequence 1, got %+v", first)
	}
	assertFileContains(t, filepath.Join(sessionDir, first.Ref), "kind: stage-stage_a-dispatched")
	assertFileContains(t, filepath.Join(sessionDir, "session.state.md"), "checkpoint_seq: 1")

	second, err := RecordCheckpoint(sessionDir, CheckpointInput{Kind: "recover-start", Source: "recovery"})
	if err != nil {
		t.Fatalf("write second checkpoint: %v", err)
	}
	if second.Sequence != 2 {
		t.Fatalf("expected second sequence 2, got %+v", second)
	}
	assertFileContains(t, filepath.Join(sessionDir, second.Ref), "kind: recover-start")
	assertFileContains(t, filepath.Join(sessionDir, "session.state.md"), "checkpoint_seq: 2")
}
