from pathlib import Path


def test_real_codex_environment_ready(real_codex_env: Path):
    assert (real_codex_env / "auth.json").exists()


def test_validate_minimal_config_success(prepare_case, run_cli):
    case_dir = prepare_case("valid-minimal")
    result = run_cli(["validate", "--config", str(case_dir / "octopus.yaml")], cwd=case_dir)
    assert result.returncode == 0, result.stderr
    assert "config is valid" in result.stdout


def test_validate_env_override_success(prepare_case, run_cli):
    case_dir = prepare_case("valid-env-override")
    result = run_cli(
        ["validate", "--config", str(case_dir / "octopus.yaml")],
        cwd=case_dir,
        env={"OPENOCTOPUS_RUNTIME__SCHEDULER__MAX_PARALLEL_ROLES": "2"},
    )
    assert result.returncode == 0, result.stderr


def test_validate_invalid_cases(prepare_case, run_cli):
    expectations = {
        "invalid-syntax": "[syntax]",
        "invalid-missing-role": "stages[0].role",
        "invalid-shell-security": "security.shell.allowlist_prefixes",
        "invalid-immutable-conflict": "forbidden_writes",
        "invalid-threshold": "runtime.master_watch.max_no_progress_rounds",
    }
    for case_name, expected in expectations.items():
        case_dir = prepare_case(case_name)
        result = run_cli(["validate", "--config", str(case_dir / "octopus.yaml")], cwd=case_dir)
        assert result.returncode != 0, case_name
        assert expected in result.stderr, result.stderr
