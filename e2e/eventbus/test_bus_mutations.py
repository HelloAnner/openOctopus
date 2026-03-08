import json
from pathlib import Path


def test_lock_conflict_and_offset_regression(prepare_module_case, run_cli, run_harness):
    case_dir = prepare_module_case("eventbus", "valid-lock-conflict")
    result = run_cli(["run", "--config", str(case_dir / "octopus.yaml")], cwd=case_dir)
    assert result.returncode == 0, result.stderr
    session_dir = parse_session_dir(result.stdout)

    lease = json.loads(
        run_harness(
            [
                "acquire-lock",
                "--session-dir",
                str(session_dir),
                "--holder",
                "orchestrator/master",
                "--ttl-seconds",
                "30",
            ],
            cwd=case_dir,
        ).stdout
    )

    first_event = json.loads(
        run_harness(
            [
                "append",
                "--session-dir",
                str(session_dir),
                "--holder",
                lease["holder"],
                "--lease-token",
                lease["lease_token"],
                "--lease-version",
                str(lease["lease_version"]),
                "--event-type",
                "SCHEDULE_UPDATED",
                "--producer",
                "orchestrator",
                "--payload-ref",
                "planner/master_schedule.md",
                "--summary",
                "schedule updated",
            ],
            cwd=case_dir,
        ).stdout
    )
    second_event = json.loads(
        run_harness(
            [
                "append",
                "--session-dir",
                str(session_dir),
                "--holder",
                lease["holder"],
                "--lease-token",
                lease["lease_token"],
                "--lease-version",
                str(lease["lease_version"]),
                "--event-type",
                "ROLE_DISPATCHED",
                "--producer",
                "orchestrator",
                "--payload-ref",
                "roles/agent_a/inbox.md",
                "--summary",
                "role dispatched",
            ],
            cwd=case_dir,
        ).stdout
    )
    assert first_event["event_id"] == "event-000002"
    assert second_event["event_id"] == "event-000003"

    commit = run_harness(
        [
            "commit-offset",
            "--session-dir",
            str(session_dir),
            "--holder",
            lease["holder"],
            "--lease-token",
            lease["lease_token"],
            "--lease-version",
            str(lease["lease_version"]),
            "--consumer-id",
            "orchestrator/master",
            "--last-event-id",
            second_event["event_id"],
            "--last-sequence",
            str(second_event["sequence"]),
            "--note",
            "schedule applied",
        ],
        cwd=case_dir,
    )
    assert commit.returncode == 0, commit.stderr

    regression = run_harness(
        [
            "commit-offset",
            "--session-dir",
            str(session_dir),
            "--holder",
            lease["holder"],
            "--lease-token",
            lease["lease_token"],
            "--lease-version",
            str(lease["lease_version"]),
            "--consumer-id",
            "orchestrator/master",
            "--last-event-id",
            first_event["event_id"],
            "--last-sequence",
            str(first_event["sequence"]),
            "--note",
            "regression",
        ],
        cwd=case_dir,
    )
    assert regression.returncode != 0
    assert "offset regression" in regression.stderr

    renewed = json.loads(
        run_harness(
            [
                "renew-lock",
                "--session-dir",
                str(session_dir),
                "--holder",
                lease["holder"],
                "--lease-token",
                lease["lease_token"],
                "--lease-version",
                str(lease["lease_version"]),
                "--ttl-seconds",
                "30",
            ],
            cwd=case_dir,
        ).stdout
    )
    stale_release = run_harness(
        [
            "release-lock",
            "--session-dir",
            str(session_dir),
            "--holder",
            lease["holder"],
            "--lease-token",
            lease["lease_token"],
            "--lease-version",
            str(lease["lease_version"]),
        ],
        cwd=case_dir,
    )
    assert stale_release.returncode != 0
    assert "lease conflict" in stale_release.stderr

    released = run_harness(
        [
            "release-lock",
            "--session-dir",
            str(session_dir),
            "--holder",
            renewed["holder"],
            "--lease-token",
            renewed["lease_token"],
            "--lease-version",
            str(renewed["lease_version"]),
        ],
        cwd=case_dir,
    )
    assert released.returncode == 0, released.stderr


def test_interrupt_lifecycle_and_chain_validation(prepare_module_case, run_cli, run_harness):
    case_dir = prepare_module_case("eventbus", "valid-minimal")
    result = run_cli(["run", "--config", str(case_dir / "octopus.yaml")], cwd=case_dir)
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

    requested = json.loads(
        run_harness(
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
                "cli",
                "--reason",
                "waiting for approval",
            ],
            cwd=case_dir,
        ).stdout
    )
    assert requested["status"] == "REQUESTED"

    acknowledged = json.loads(
        run_harness(
            [
                "ack-interrupt",
                "--session-dir",
                str(session_dir),
                "--holder",
                lease["holder"],
                "--lease-token",
                lease["lease_token"],
                "--lease-version",
                str(lease["lease_version"]),
                "--interrupt-id",
                requested["interrupt_id"],
            ],
            cwd=case_dir,
        ).stdout
    )
    assert acknowledged["status"] == "ACKNOWLEDGED"

    cleared = json.loads(
        run_harness(
            [
                "clear-interrupt",
                "--session-dir",
                str(session_dir),
                "--holder",
                lease["holder"],
                "--lease-token",
                lease["lease_token"],
                "--lease-version",
                str(lease["lease_version"]),
                "--interrupt-id",
                requested["interrupt_id"],
            ],
            cwd=case_dir,
        ).stdout
    )
    assert cleared["status"] == "CLEARED"

    interrupts_file = (session_dir / "bus" / "interrupts.md").read_text()
    assert "CLEARED" in interrupts_file

    events_path = session_dir / "bus" / "events.md"
    broken = events_path.read_text().replace("- prev_event_hash: ", "- prev_event_hash: sha256:broken-", 1)
    events_path.write_text(broken)

    listing = run_harness(["list", "--session-dir", str(session_dir)], cwd=case_dir)
    assert listing.returncode != 0
    assert "event chain broken" in listing.stderr


def parse_session_dir(stdout: str) -> Path:
    prefix = "session created: "
    for line in stdout.splitlines():
        if line.startswith(prefix):
            return Path(line[len(prefix) :].strip())
    raise AssertionError(f"session path not found in stdout: {stdout}")
