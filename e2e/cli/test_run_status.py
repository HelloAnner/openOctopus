import json


def test_run_json_success(prepare_module_case, run_cli):
    case_dir = prepare_module_case("cli", "valid-minimal")

    result = run_cli(["run", "--config", str(case_dir / "octopus.yaml"), "--format", "json"], cwd=case_dir)

    assert result.returncode == 0, result.stderr
    payload = json.loads(result.stdout)
    assert payload["ok"] is True
    assert payload["command"] == "run"
    assert payload["data"]["session_id"]
    assert payload["data"]["session_dir"]


def test_status_json_reads_completed_session(prepare_module_case, run_cli):
    case_dir = prepare_module_case("cli", "valid-deterministic-success")
    run_result = run_cli(
        ["run", "--config", str(case_dir / "octopus.yaml"), "--format", "json"],
        cwd=case_dir,
        env={
            "OPENOCTOPUS_DISABLE_ROLE_RUNTIME_LOOP": "0",
            "OPENOCTOPUS_DETERMINISTIC_RESULTS_AGENT_A": "SUCCESS",
        },
    )
    assert run_result.returncode == 0, run_result.stderr
    session_dir = json.loads(run_result.stdout)["data"]["session_dir"]

    status_result = run_cli(["status", "--session", session_dir, "--format", "json"], cwd=case_dir)

    assert status_result.returncode == 0, status_result.stderr
    payload = json.loads(status_result.stdout)
    assert payload["ok"] is True
    assert payload["command"] == "status"
    assert payload["data"]["workflow_status"] == "COMPLETED"
    assert payload["data"]["current_stage_id"] == "stage_a"
    assert payload["data"]["schedule_version"] >= 1


def test_interrupt_all_then_status_reports_waiting_human(prepare_module_case, run_cli):
    case_dir = prepare_module_case("cli", "valid-interrupt-all")
    run_result = run_cli(["run", "--config", str(case_dir / "octopus.yaml"), "--format", "json"], cwd=case_dir)
    assert run_result.returncode == 0, run_result.stderr
    session_dir = json.loads(run_result.stdout)["data"]["session_dir"]

    interrupt_result = run_cli(
        ["interrupt-all", "--session", session_dir, "--reason", "manual review", "--format", "json"],
        cwd=case_dir,
    )
    assert interrupt_result.returncode == 0, interrupt_result.stderr
    assert json.loads(interrupt_result.stdout)["data"]["requested_count"] >= 2

    status_result = run_cli(["status", "--session", session_dir, "--format", "json"], cwd=case_dir)

    assert status_result.returncode == 0, status_result.stderr
    payload = json.loads(status_result.stdout)
    assert payload["data"]["workflow_status"] == "WAITING_HUMAN"
    assert payload["data"]["blocker_summary"] == "manual review"


def test_status_missing_session_returns_stable_error(prepare_module_case, run_cli):
    case_dir = prepare_module_case("cli", "valid-minimal")

    result = run_cli(["status", "--session", "missing", "--format", "json"], cwd=case_dir)

    assert result.returncode == 3, result.stderr
    payload = json.loads(result.stderr)
    assert payload["ok"] is False
    assert payload["command"] == "status"
    assert payload["error"]["code"] == "session_not_found"
