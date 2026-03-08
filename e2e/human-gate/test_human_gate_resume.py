from pathlib import Path


def test_inject_and_resume_complete_interrupted_session(prepare_module_case, run_cli, run_role_runtime_harness):
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

    ack_tick = run_role_runtime_harness(
        ["tick-role", "--session-dir", str(session_dir), "--role-id", "agent_a"],
        cwd=case_dir,
    )
    assert ack_tick.returncode == 0, ack_tick.stderr

    inject_result = run_cli(
        ["inject", "--session", str(session_dir), "--role", "agent_a", "--message", "继续执行 stage_a"],
        cwd=case_dir,
    )
    assert inject_result.returncode == 0, inject_result.stderr

    resume_result = run_cli(
        ["resume", "--session", str(session_dir), "--role", "agent_a"],
        cwd=case_dir,
        env={
            "OPENOCTOPUS_DISABLE_ROLE_RUNTIME_LOOP": "0",
            "OPENOCTOPUS_DETERMINISTIC_RESULTS_AGENT_A": "SUCCESS",
        },
    )
    assert resume_result.returncode == 0, resume_result.stderr

    assert "status: CLEARED" in (session_dir / "bus" / "interrupts.md").read_text()
    assert "message_id: msg-000001" in (session_dir / "planner" / "human_messages.md").read_text()
    assert "human_message_cursor: msg-000001" in (session_dir / "planner" / "requirement.snapshot.md").read_text()
    assert (session_dir / "roles" / "agent_a" / "turns" / "0001-output.md").exists()
    assert "status: COMPLETED" in (session_dir / "session.state.md").read_text()


def test_resume_requeues_blocked_session(prepare_module_case, run_cli):
    case_dir = prepare_module_case("human-gate", "valid-blocked-resume")
    run_result = run_cli(
        ["run", "--config", str(case_dir / "octopus.yaml")],
        cwd=case_dir,
        env={
            "OPENOCTOPUS_DISABLE_ROLE_RUNTIME_LOOP": "0",
            "OPENOCTOPUS_DETERMINISTIC_RESULTS_AGENT_A": "BLOCKED",
        },
    )
    assert run_result.returncode == 0, run_result.stderr
    session_dir = parse_session_dir(run_result.stdout)

    assert "status: WAITING_HUMAN" in (session_dir / "session.state.md").read_text()
    assert "deterministic result BLOCKED" in (session_dir / "planner" / "blockers.md").read_text()

    inject_result = run_cli(
        ["inject", "--session", str(session_dir), "--message", "补充约束后继续"],
        cwd=case_dir,
    )
    assert inject_result.returncode == 0, inject_result.stderr

    resume_result = run_cli(
        ["resume", "--session", str(session_dir)],
        cwd=case_dir,
        env={
            "OPENOCTOPUS_DISABLE_ROLE_RUNTIME_LOOP": "0",
            "OPENOCTOPUS_DETERMINISTIC_RESULTS_AGENT_A": "SUCCESS",
        },
    )
    assert resume_result.returncode == 0, resume_result.stderr

    turns_dir = session_dir / "roles" / "agent_a" / "turns"
    assert (turns_dir / "0002-input.md").exists()
    assert (turns_dir / "0002-output.md").exists()
    assert "task_id: task-stage_a-02" in (session_dir / "roles" / "agent_a" / "inbox.md").read_text()
    assert "status: SUCCESS" in (session_dir / "roles" / "agent_a" / "conclusion.md").read_text()
    assert "status: COMPLETED" in (session_dir / "session.state.md").read_text()


def parse_session_dir(stdout: str) -> Path:
    prefix = "session created: "
    for line in stdout.splitlines():
        if line.startswith(prefix):
            return Path(line[len(prefix) :].strip())
    raise AssertionError(f"session path not found in stdout: {stdout}")
