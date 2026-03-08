import json
from pathlib import Path


def test_interrupt_before_role_tick(prepare_module_case, run_cli, run_harness, run_role_runtime_harness):
    case_dir = prepare_module_case("role-runtime", "valid-interrupt-before-start")
    result = run_cli(
        ["run", "--config", str(case_dir / "octopus.yaml")],
        cwd=case_dir,
        env={"OPENOCTOPUS_DISABLE_ROLE_RUNTIME_LOOP": "1"},
    )

    assert result.returncode == 0, result.stderr
    session_dir = parse_session_dir(result.stdout)

    lease = json.loads(
        run_harness(
            [
                "acquire-lock",
                "--session-dir",
                str(session_dir),
                "--holder",
                "human-gate",
                "--ttl-seconds",
                "30",
            ],
            cwd=case_dir,
        ).stdout
    )

    interrupted = run_harness(
        [
            "request-interrupt",
            "--session-dir",
            str(session_dir),
            "--holder",
            lease["holder"],
            "--lease-token",
            lease["lease_token"],
            "--lease-version",
            str(lease["lease_version"]),
            "--scope",
            "role",
            "--target-role-id",
            "agent_a",
            "--source",
            "e2e",
            "--reason",
            "stop before role tick",
        ],
        cwd=case_dir,
    )
    assert interrupted.returncode == 0, interrupted.stderr

    tick = run_role_runtime_harness(
        ["tick-role", "--session-dir", str(session_dir), "--role-id", "agent_a"],
        cwd=case_dir,
        env={"OPENOCTOPUS_DETERMINISTIC_RESULTS_AGENT_A": "SUCCESS"},
    )
    assert tick.returncode == 0, tick.stderr
    assert "status: INTERRUPTED" in (session_dir / "roles" / "agent_a" / "state.md").read_text()
    assert not (session_dir / "roles" / "agent_a" / "turns" / "0001-output.md").exists()


def test_reset_increments_generation_and_keeps_turns(prepare_module_case, run_cli, run_role_runtime_harness):
    case_dir = prepare_module_case("role-runtime", "valid-reset-generation")
    result = run_cli(
        ["run", "--config", str(case_dir / "octopus.yaml")],
        cwd=case_dir,
        env={"OPENOCTOPUS_DETERMINISTIC_RESULTS_AGENT_A": "SUCCESS"},
    )

    assert result.returncode == 0, result.stderr
    session_dir = parse_session_dir(result.stdout)
    assert (session_dir / "roles" / "agent_a" / "turns" / "0001-output.md").exists()

    written = run_role_runtime_harness(
        [
            "write-reset",
            "--session-dir",
            str(session_dir),
            "--role-id",
            "agent_a",
            "--reason",
            "clear context",
            "--requested-by",
            "e2e",
        ],
        cwd=case_dir,
    )
    assert written.returncode == 0, written.stderr

    tick = run_role_runtime_harness(
        ["tick-role", "--session-dir", str(session_dir), "--role-id", "agent_a"],
        cwd=case_dir,
    )
    assert tick.returncode == 0, tick.stderr
    assert "session_generation: 2" in (session_dir / "roles" / "agent_a" / "state.md").read_text()
    assert (session_dir / "roles" / "agent_a" / "turns" / "0001-output.md").exists()


def parse_session_dir(stdout: str) -> Path:
    prefix = "session created: "
    for line in stdout.splitlines():
        if line.startswith(prefix):
            return Path(line[len(prefix) :].strip())
    raise AssertionError(f"session path not found in stdout: {stdout}")
