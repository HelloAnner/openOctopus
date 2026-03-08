from pathlib import Path


def test_run_executes_deterministic_role_runtime(prepare_module_case, run_cli):
    case_dir = prepare_module_case("role-runtime", "valid-deterministic-success")
    result = run_cli(
        ["run", "--config", str(case_dir / "octopus.yaml")],
        cwd=case_dir,
        env={"OPENOCTOPUS_DISABLE_ROLE_RUNTIME_LOOP": "0", "OPENOCTOPUS_DETERMINISTIC_RESULTS_AGENT_A": "SUCCESS"},
    )

    assert result.returncode == 0, result.stderr
    session_dir = parse_session_dir(result.stdout)
    assert (session_dir / "roles" / "agent_a" / "turns" / "0001-input.md").exists()
    assert (session_dir / "roles" / "agent_a" / "turns" / "0001-output.md").exists()
    assert "status: SUCCESS" in (session_dir / "roles" / "agent_a" / "conclusion.md").read_text()
    assert "status: COMPLETED" in (session_dir / "session.state.md").read_text()


def test_run_retries_deterministic_role_runtime(prepare_module_case, run_cli):
    case_dir = prepare_module_case("role-runtime", "valid-retry-once")
    result = run_cli(
        ["run", "--config", str(case_dir / "octopus.yaml")],
        cwd=case_dir,
        env={"OPENOCTOPUS_DISABLE_ROLE_RUNTIME_LOOP": "0", "OPENOCTOPUS_DETERMINISTIC_RESULTS_AGENT_A": "NEEDS_RETRY,SUCCESS"},
    )

    assert result.returncode == 0, result.stderr
    session_dir = parse_session_dir(result.stdout)
    turns_dir = session_dir / "roles" / "agent_a" / "turns"
    assert (turns_dir / "0001-input.md").exists()
    assert (turns_dir / "0001-output.md").exists()
    assert (turns_dir / "0002-input.md").exists()
    assert (turns_dir / "0002-output.md").exists()
    assert "status: SUCCESS" in (session_dir / "roles" / "agent_a" / "conclusion.md").read_text()


def test_run_blocked_role_runtime_waits_human(prepare_module_case, run_cli):
    case_dir = prepare_module_case("role-runtime", "valid-blocked")
    result = run_cli(
        ["run", "--config", str(case_dir / "octopus.yaml")],
        cwd=case_dir,
        env={"OPENOCTOPUS_DISABLE_ROLE_RUNTIME_LOOP": "0", "OPENOCTOPUS_DETERMINISTIC_RESULTS_AGENT_A": "BLOCKED"},
    )

    assert result.returncode == 0, result.stderr
    session_dir = parse_session_dir(result.stdout)
    assert "status: BLOCKED" in (session_dir / "roles" / "agent_a" / "conclusion.md").read_text()
    assert "status: WAITING_HUMAN" in (session_dir / "session.state.md").read_text()
    assert "deterministic result BLOCKED" in (session_dir / "planner" / "blockers.md").read_text()


def parse_session_dir(stdout: str) -> Path:
    prefix = "session created: "
    for line in stdout.splitlines():
        if line.startswith(prefix):
            return Path(line[len(prefix) :].strip())
    raise AssertionError(f"session path not found in stdout: {stdout}")
