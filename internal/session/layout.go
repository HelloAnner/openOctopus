package session

import "path/filepath"

var requiredDirectories = []string{
	"planner",
	"bus",
	"roles",
	"artifacts",
	"state",
	filepath.Join("state", "checkpoints"),
	"audit",
}

var placeholderFiles = map[string]string{
	filepath.Join("planner", "requirement.snapshot.md"): "Requirement Snapshot",
	filepath.Join("planner", "human_messages.md"):       "Human Messages",
	filepath.Join("planner", "master_schedule.md"):      "Master Schedule",
	filepath.Join("planner", "global_progress.md"):      "Global Progress",
	filepath.Join("planner", "blockers.md"):             "Blockers",
	filepath.Join("bus", "events.md"):                   "Bus Events",
	filepath.Join("bus", "interrupts.md"):               "Interrupts",
	filepath.Join("bus", "offsets.md"):                  "Offsets",
	filepath.Join("bus", "lock.md"):                     "Lock",
	filepath.Join("artifacts", "index.md"):              "Artifacts Index",
	filepath.Join("audit", "lineage.md"):                "Lineage",
	filepath.Join("audit", "replay.md"):                 "Replay",
}
