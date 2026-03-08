def test_run_valid_config_creates_session(prepare_case, run_cli):
    case_dir = prepare_case("valid-minimal")
    result = run_cli(["run", "--config", str(case_dir / "octopus.yaml")], cwd=case_dir)
    assert result.returncode == 0, result.stderr
    sessions_dir = case_dir / ".octopus" / "sessions"
    assert sessions_dir.exists()
    session_dirs = list(sessions_dir.iterdir())
    assert len(session_dirs) == 1
    assert (session_dirs[0] / "metadata.md").exists()


def test_run_invalid_config_does_not_create_session(prepare_case, run_cli):
    case_dir = prepare_case("invalid-missing-role")
    result = run_cli(["run", "--config", str(case_dir / "octopus.yaml")], cwd=case_dir)
    assert result.returncode != 0
    assert not (case_dir / ".octopus" / "sessions").exists()
