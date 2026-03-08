import json
from pathlib import Path


def test_recover_continues_dispatched_session(prepare_module_case, run_cli):
    case_dir = prepare_module_case("recovery", "valid-dispatched-session")
    run_result = run_cli(
        ["run", "--config", str(case_dir / "octopus.yaml")],
        cwd=case_dir,
        env={"OPENOCTOPUS_DISABLE_ROLE_RUNTIME_LOOP": "1"},
    )
    assert run_result.returncode == 0, run_result.stderr
    session_dir = parse_session_dir(run_result.stdout)

    recover_result = run_cli(
        ["recover", "--session", str(session_dir), "--format", "json"],
        cwd=case_dir,
        env={
            "OPENOCTOPUS_DISABLE_ROLE_RUNTIME_LOOP": "0",
            "OPENOCTOPUS_DETERMINISTIC_RESULTS_AGENT_A": "SUCCESS",
        },
    )
    assert recover_result.returncode == 0, recover_result.stderr
    payload = json.loads(recover_result.stdout)

    assert payload["data"]["continued"] is True
    assert payload["data"]["recovered_status"] == "COMPLETED"
    assert (session_dir / "roles" / "agent_a" / "turns" / "0001-output.md").exists()
    assert "status: COMPLETED" in (session_dir / "session.state.md").read_text()
    assert "continued: true" in (session_dir / "audit" / "replay.md").read_text()


def test_recover_repairs_missing_session_state(prepare_module_case, run_cli):
    case_dir = prepare_module_case("recovery", "valid-missing-session-state")
    run_result = run_cli(
        ["run", "--config", str(case_dir / "octopus.yaml")],
        cwd=case_dir,
        env={"OPENOCTOPUS_DISABLE_ROLE_RUNTIME_LOOP": "1"},
    )
    assert run_result.returncode == 0, run_result.stderr
    session_dir = parse_session_dir(run_result.stdout)

    (session_dir / "session.state.md").unlink()

    recover_result = run_cli(
        ["recover", "--session", str(session_dir), "--format", "json"],
        cwd=case_dir,
        env={
            "OPENOCTOPUS_DISABLE_ROLE_RUNTIME_LOOP": "0",
            "OPENOCTOPUS_DETERMINISTIC_RESULTS_AGENT_A": "SUCCESS",
        },
    )
    assert recover_result.returncode == 0, recover_result.stderr
    payload = json.loads(recover_result.stdout)

    assert payload["data"]["continued"] is True
    assert payload["data"]["recovered_status"] == "COMPLETED"
    assert (session_dir / "session.state.md").exists()
    session_state = (session_dir / "session.state.md").read_text()
    assert "status: COMPLETED" in session_state
    assert "checkpoint_seq:" in session_state


def test_recover_fails_for_broken_event_chain(prepare_module_case, run_cli):
    case_dir = prepare_module_case("recovery", "valid-broken-events")
    run_result = run_cli(
        ["run", "--config", str(case_dir / "octopus.yaml")],
        cwd=case_dir,
        env={"OPENOCTOPUS_DISABLE_ROLE_RUNTIME_LOOP": "1"},
    )
    assert run_result.returncode == 0, run_result.stderr
    session_dir = parse_session_dir(run_result.stdout)

    events_path = session_dir / "bus" / "events.md"
    content = events_path.read_text()
    events_path.write_text(content.replace("- event_hash: ", "- event_hash: broken-", 1))

    recover_result = run_cli(["recover", "--session", str(session_dir), "--format", "json"], cwd=case_dir)

    assert recover_result.returncode == 4, recover_result.stderr
    payload = json.loads(recover_result.stderr)
    assert payload["ok"] is False
    assert payload["error"]["code"] == "event_chain_broken"
    assert not (session_dir / "roles" / "agent_a" / "turns" / "0001-output.md").exists()


def test_recover_waiting_human_without_continue(prepare_module_case, run_cli):
    case_dir = prepare_module_case("recovery", "valid-waiting-human")
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

    recover_result = run_cli(["recover", "--session", str(session_dir), "--format", "json"], cwd=case_dir)
    assert recover_result.returncode == 0, recover_result.stderr
    payload = json.loads(recover_result.stdout)

    assert payload["data"]["continued"] is False
    assert payload["data"]["recovered_status"] == "WAITING_HUMAN"
    assert payload["data"]["reason"] == "needs_human_resume"
    assert "status: WAITING_HUMAN" in (session_dir / "session.state.md").read_text()
    assert not (session_dir / "roles" / "agent_a" / "turns" / "0002-output.md").exists()


def parse_session_dir(stdout: str) -> Path:
    prefix = "session created: "
    for line in stdout.splitlines():
        if line.startswith(prefix):
            return Path(line[len(prefix) :].strip())
    raise AssertionError(f"session path not found in stdout: {stdout}")
