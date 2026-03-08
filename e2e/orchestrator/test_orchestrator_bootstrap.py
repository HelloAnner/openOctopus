from pathlib import Path


def test_run_bootstraps_orchestrator(prepare_module_case, run_cli):
    case_dir = prepare_module_case("orchestrator", "valid-minimal")
    result = run_cli(["run", "--config", str(case_dir / "octopus.yaml")], cwd=case_dir)

    assert result.returncode == 0, result.stderr
    session_dir = parse_session_dir(result.stdout)
    assert session_dir.exists()

    schedule = (session_dir / "planner" / "master_schedule.md").read_text()
    assert "Initialized by session 001." not in schedule
    assert "stage_id: stage_a" in schedule

    assert (session_dir / "planner" / "task_board.md").exists()
    assert (session_dir / "planner" / "task_graph.mmd").exists()
    assert (session_dir / "planner" / "dispatch_log.md").exists()
    assert (session_dir / "planner" / "decision_log.md").exists()

    context = (session_dir / "roles" / "agent_a" / "context.md").read_text()
    inbox = (session_dir / "roles" / "agent_a" / "inbox.md").read_text()
    assert "task_id: task-stage_a-01" in context
    assert "task_id: task-stage_a-01" in inbox


def parse_session_dir(stdout: str) -> Path:
    prefix = "session created: "
    for line in stdout.splitlines():
        if line.startswith(prefix):
            return Path(line[len(prefix) :].strip())
    raise AssertionError(f"session path not found in stdout: {stdout}")
