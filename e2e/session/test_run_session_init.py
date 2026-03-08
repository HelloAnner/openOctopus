from pathlib import Path


def test_run_creates_full_session_skeleton(prepare_module_case, run_cli):
    case_dir = prepare_module_case("session", "valid-minimal")
    result = run_cli(["run", "--config", str(case_dir / "octopus.yaml")], cwd=case_dir)

    assert result.returncode == 0, result.stderr
    session_dir = parse_session_dir(result.stdout)
    assert session_dir.exists()

    required_paths = [
        session_dir / "metadata.md",
        session_dir / "session.state.md",
        session_dir / "timeline.md",
        session_dir / "planner" / "human_messages.md",
        session_dir / "bus" / "events.md",
        session_dir / "artifacts" / "index.md",
        session_dir / "state" / "effective_config.yaml",
        session_dir / "state" / "checkpoints" / "0000-init.md",
        session_dir / "audit" / "lineage.md",
    ]
    for item in required_paths:
        assert item.exists(), f"missing path: {item}"

    metadata = (session_dir / "metadata.md").read_text()
    assert "- status: INITIAL" in metadata
    assert "- applied_defaults_count:" in metadata

    timeline = (session_dir / "timeline.md").read_text()
    assert "SESSION_CREATED" in timeline


def test_run_honors_custom_sessions_dir(prepare_module_case, run_cli):
    case_dir = prepare_module_case("session", "valid-custom-sessions-dir")
    result = run_cli(["run", "--config", str(case_dir / "octopus.yaml")], cwd=case_dir)

    assert result.returncode == 0, result.stderr
    session_dir = parse_session_dir(result.stdout)
    expected_root = case_dir / ".octopus-custom" / "sessions-home"
    assert session_dir.parent == expected_root
    assert session_dir.exists()


def parse_session_dir(stdout: str) -> Path:
    prefix = "session created: "
    for line in stdout.splitlines():
        if line.startswith(prefix):
            return Path(line[len(prefix) :].strip())
    raise AssertionError(f"session path not found in stdout: {stdout}")
