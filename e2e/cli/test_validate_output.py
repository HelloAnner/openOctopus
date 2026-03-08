import json


def test_validate_json_success(prepare_module_case, run_cli):
    case_dir = prepare_module_case("cli", "valid-minimal")

    result = run_cli(["validate", "--config", str(case_dir / "octopus.yaml"), "--format", "json"], cwd=case_dir)

    assert result.returncode == 0, result.stderr
    payload = json.loads(result.stdout)
    assert payload["ok"] is True
    assert payload["command"] == "validate"
    assert payload["data"]["applied_defaults_count"] > 0


def test_validate_json_failure_returns_stable_error(prepare_module_case, run_cli):
    case_dir = prepare_module_case("cli", "valid-minimal")
    invalid_config = case_dir / "invalid-octopus.yaml"
    invalid_config.write_text(
        """version: \"2.1\"\n\nmeta:\n  workflow_id: \"cli-invalid\"\n  name: \"CLI Invalid\"\n\nllm_profiles:\n  codex_cli:\n    provider: \"codex\"\n    mode: \"cli\"\n    cli_path: \"codex\"\n\ntool_registry:\n  builtin:\n    file_read:\n      module: \"openoctopus.tools.file\"\n      class: \"FileReadTool\"\n\nroles:\n  - id: \"agent_a\"\n    name: \"Agent A\"\n    type: \"react\"\n    llm_profile: \"codex_cli\"\n    system_prompt: \"你负责执行任务。\"\n    tools: [\"file_read\"]\n\nstages:\n  - id: \"stage_a\"\n    name: \"Stage A\"\n    role: \"missing_role\"\n    output:\n      - type: \"artifact\"\n        name: \"artifact_a\"\n\ntransitions:\n  - from: \"stage_a\"\n    to: \"__END__\"\n"""
    )

    result = run_cli(["validate", "--config", str(invalid_config), "--format", "json"], cwd=case_dir)

    assert result.returncode == 2, result.stderr
    payload = json.loads(result.stderr)
    assert payload["ok"] is False
    assert payload["command"] == "validate"
    assert payload["error"]["code"] == "config_validation_failed"

