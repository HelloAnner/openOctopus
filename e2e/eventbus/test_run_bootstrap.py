import json
from pathlib import Path


def test_run_bootstraps_event_bus(prepare_module_case, run_cli):
    case_dir = prepare_module_case("eventbus", "valid-minimal")
    result = run_cli(["run", "--config", str(case_dir / "octopus.yaml")], cwd=case_dir)

    assert result.returncode == 0, result.stderr
    session_dir = parse_session_dir(result.stdout)
    assert session_dir.exists()

    events_file = (session_dir / "bus" / "events.md").read_text()
    assert "SESSION_CREATED" in events_file
    assert "event-000001" in events_file
    assert "Initialized by session 001." not in events_file

    lock_file = (session_dir / "bus" / "lock.md").read_text()
    assert "- status: FREE" in lock_file
    assert "Initialized by session 001." not in lock_file


def test_bootstrap_is_idempotent(prepare_module_case, run_cli, run_harness):
    case_dir = prepare_module_case("eventbus", "valid-repeat-bootstrap")
    result = run_cli(["run", "--config", str(case_dir / "octopus.yaml")], cwd=case_dir)

    assert result.returncode == 0, result.stderr
    session_dir = parse_session_dir(result.stdout)

    harness = run_harness(["bootstrap", "--session-dir", str(session_dir)], cwd=case_dir)
    assert harness.returncode == 0, harness.stderr
    payload = json.loads(harness.stdout)
    assert payload["status"] == "ok"

    events_file = (session_dir / "bus" / "events.md").read_text()
    assert events_file.count("SESSION_CREATED") == 1


def parse_session_dir(stdout: str) -> Path:
    prefix = "session created: "
    for line in stdout.splitlines():
        if line.startswith(prefix):
            return Path(line[len(prefix) :].strip())
    raise AssertionError(f"session path not found in stdout: {stdout}")
