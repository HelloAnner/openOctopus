from pathlib import Path


def test_tick_applies_success_conclusion(prepare_module_case, run_cli, run_orchestrator_harness):
    case_dir = prepare_module_case("orchestrator", "valid-minimal")
    result = run_cli(["run", "--config", str(case_dir / "octopus.yaml")], cwd=case_dir)

    assert result.returncode == 0, result.stderr
    session_dir = parse_session_dir(result.stdout)

    written = run_orchestrator_harness(
        [
            "write-conclusion",
            "--session-dir",
            str(session_dir),
            "--role-id",
            "agent_a",
            "--stage-id",
            "stage_a",
            "--task-id",
            "task-stage_a-01",
            "--status",
            "SUCCESS",
            "--summary",
            "done",
        ],
        cwd=case_dir,
    )
    assert written.returncode == 0, written.stderr

    tick = run_orchestrator_harness(["tick", "--session-dir", str(session_dir)], cwd=case_dir)
    assert tick.returncode == 0, tick.stderr

    state = (session_dir / "session.state.md").read_text()
    schedule = (session_dir / "planner" / "master_schedule.md").read_text()
    assert "status: COMPLETED" in state
    assert "status: COMPLETED" in schedule


def parse_session_dir(stdout: str) -> Path:
    prefix = "session created: "
    for line in stdout.splitlines():
        if line.startswith(prefix):
            return Path(line[len(prefix) :].strip())
    raise AssertionError(f"session path not found in stdout: {stdout}")
