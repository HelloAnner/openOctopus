from pathlib import Path


def test_retry_blocked_human_message_and_parallel_limit(prepare_module_case, run_cli, run_orchestrator_harness):
    retry_case = prepare_module_case("orchestrator", "valid-minimal")
    retry_result = run_cli(["run", "--config", str(retry_case / "octopus.yaml")], cwd=retry_case)
    assert retry_result.returncode == 0, retry_result.stderr
    retry_session_dir = parse_session_dir(retry_result.stdout)

    written_retry = run_orchestrator_harness(
        [
            "write-conclusion",
            "--session-dir",
            str(retry_session_dir),
            "--role-id",
            "agent_a",
            "--stage-id",
            "stage_a",
            "--task-id",
            "task-stage_a-01",
            "--status",
            "NEEDS_RETRY",
            "--summary",
            "retry me",
        ],
        cwd=retry_case,
    )
    assert written_retry.returncode == 0, written_retry.stderr
    retry_tick = run_orchestrator_harness(["tick", "--session-dir", str(retry_session_dir)], cwd=retry_case)
    assert retry_tick.returncode == 0, retry_tick.stderr
    retry_schedule = (retry_session_dir / "planner" / "master_schedule.md").read_text()
    retry_inbox = (retry_session_dir / "roles" / "agent_a" / "inbox.md").read_text()
    assert "- attempt: 2" in retry_schedule
    assert "task_id: task-stage_a-02" in retry_inbox

    blocked_case = prepare_module_case("orchestrator", "valid-minimal")
    blocked_result = run_cli(["run", "--config", str(blocked_case / "octopus.yaml")], cwd=blocked_case)
    assert blocked_result.returncode == 0, blocked_result.stderr
    blocked_session_dir = parse_session_dir(blocked_result.stdout)
    written_blocked = run_orchestrator_harness(
        [
            "write-conclusion",
            "--session-dir",
            str(blocked_session_dir),
            "--role-id",
            "agent_a",
            "--stage-id",
            "stage_a",
            "--task-id",
            "task-stage_a-01",
            "--status",
            "BLOCKED",
            "--summary",
            "need human",
        ],
        cwd=blocked_case,
    )
    assert written_blocked.returncode == 0, written_blocked.stderr
    blocked_tick = run_orchestrator_harness(["tick", "--session-dir", str(blocked_session_dir)], cwd=blocked_case)
    assert blocked_tick.returncode == 0, blocked_tick.stderr
    blockers = (blocked_session_dir / "planner" / "blockers.md").read_text()
    state = (blocked_session_dir / "session.state.md").read_text()
    assert "need human" in blockers
    assert "status: WAITING_HUMAN" in state

    message_case = prepare_module_case("orchestrator", "valid-human-message")
    message_result = run_cli(["run", "--config", str(message_case / "octopus.yaml")], cwd=message_case)
    assert message_result.returncode == 0, message_result.stderr
    message_session_dir = parse_session_dir(message_result.stdout)
    appended = run_orchestrator_harness(
        [
            "append-human-message",
            "--session-dir",
            str(message_session_dir),
            "--source",
            "user",
            "--message",
            "继续推进 orchestrator",
        ],
        cwd=message_case,
    )
    assert appended.returncode == 0, appended.stderr
    message_tick = run_orchestrator_harness(["tick", "--session-dir", str(message_session_dir)], cwd=message_case)
    assert message_tick.returncode == 0, message_tick.stderr
    snapshot = (message_session_dir / "planner" / "requirement.snapshot.md").read_text()
    assert "snapshot_version: 2" in snapshot
    assert "human_message_cursor: msg-000001" in snapshot

    parallel_case = prepare_module_case("orchestrator", "valid-two-entry")
    parallel_result = run_cli(["run", "--config", str(parallel_case / "octopus.yaml")], cwd=parallel_case)
    assert parallel_result.returncode == 0, parallel_result.stderr
    parallel_session_dir = parse_session_dir(parallel_result.stdout)
    parallel_schedule = (parallel_session_dir / "planner" / "master_schedule.md").read_text()
    assert parallel_schedule.count("status: DISPATCHED") == 1


def parse_session_dir(stdout: str) -> Path:
    prefix = "session created: "
    for line in stdout.splitlines():
        if line.startswith(prefix):
            return Path(line[len(prefix) :].strip())
    raise AssertionError(f"session path not found in stdout: {stdout}")
