from pathlib import Path


def test_interrupt_ack_blocks_role_until_cleared(prepare_module_case, run_cli, run_role_runtime_harness):
    case_dir = prepare_module_case("human-gate", "valid-interrupt-resume")
    run_result = run_cli(
        ["run", "--config", str(case_dir / "octopus.yaml")],
        cwd=case_dir,
        env={"OPENOCTOPUS_DISABLE_ROLE_RUNTIME_LOOP": "1"},
    )
    assert run_result.returncode == 0, run_result.stderr
    session_dir = parse_session_dir(run_result.stdout)

    interrupt_result = run_cli(
        ["interrupt", "--session", str(session_dir), "--role", "agent_a", "--reason", "manual review"],
        cwd=case_dir,
    )
    assert interrupt_result.returncode == 0, interrupt_result.stderr

    first_tick = run_role_runtime_harness(
        ["tick-role", "--session-dir", str(session_dir), "--role-id", "agent_a"],
        cwd=case_dir,
    )
    assert first_tick.returncode == 0, first_tick.stderr
    assert "status: INTERRUPTED" in (session_dir / "roles" / "agent_a" / "state.md").read_text()

    turns_dir = session_dir / "roles" / "agent_a" / "turns"
    assert list(turns_dir.iterdir()) == []

    second_tick = run_role_runtime_harness(
        ["tick-role", "--session-dir", str(session_dir), "--role-id", "agent_a"],
        cwd=case_dir,
    )
    assert second_tick.returncode == 0, second_tick.stderr
    assert list(turns_dir.iterdir()) == []
    assert "status: ACKNOWLEDGED" in (session_dir / "bus" / "interrupts.md").read_text()


def test_interrupt_all_marks_waiting_human(prepare_module_case, run_cli):
    case_dir = prepare_module_case("human-gate", "valid-interrupt-all")
    run_result = run_cli(
        ["run", "--config", str(case_dir / "octopus.yaml")],
        cwd=case_dir,
        env={"OPENOCTOPUS_DISABLE_ROLE_RUNTIME_LOOP": "1"},
    )
    assert run_result.returncode == 0, run_result.stderr
    session_dir = parse_session_dir(run_result.stdout)

    interrupt_result = run_cli(
        ["interrupt-all", "--session", str(session_dir), "--reason", "manual review"],
        cwd=case_dir,
    )
    assert interrupt_result.returncode == 0, interrupt_result.stderr

    assert "status: WAITING_HUMAN" in (session_dir / "session.state.md").read_text()
    assert "manual review" in (session_dir / "planner" / "blockers.md").read_text()
    interrupts = (session_dir / "bus" / "interrupts.md").read_text()
    assert interrupts.count("status: REQUESTED") >= 2


def parse_session_dir(stdout: str) -> Path:
    prefix = "session created: "
    for line in stdout.splitlines():
        if line.startswith(prefix):
            return Path(line[len(prefix) :].strip())
    raise AssertionError(f"session path not found in stdout: {stdout}")
