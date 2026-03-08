from pathlib import Path


def test_run_publishes_artifact_version(prepare_module_case, run_cli):
    case_dir = prepare_module_case("artifact", "valid-deterministic-publish")
    source = case_dir / "sources" / "artifact_a.md"
    result = run_cli(
        ["run", "--config", str(case_dir / "octopus.yaml")],
        cwd=case_dir,
        env={
            "OPENOCTOPUS_DISABLE_ROLE_RUNTIME_LOOP": "0",
            "OPENOCTOPUS_DETERMINISTIC_RESULTS_AGENT_A": "SUCCESS",
            "OPENOCTOPUS_DETERMINISTIC_OUTPUT_REFS_AGENT_A": str(source),
        },
    )

    assert result.returncode == 0, result.stderr
    session_dir = parse_session_dir(result.stdout)
    index_text = (session_dir / "artifacts" / "index.md").read_text()
    assert "# Artifact Index" in index_text
    assert "artifact_name: artifact_a" in index_text
    assert (session_dir / "artifacts" / "artifact_a" / "0001" / "manifest.md").exists()
    assert (session_dir / "artifacts" / "artifact_a" / "0001" / "diff.md").exists()
    assert (session_dir / "artifacts" / "artifact_a" / "0001" / "content.md").exists()
    assert "record_count: 1" in (session_dir / "audit" / "lineage.md").read_text()
    assert "status: COMPLETED" in (session_dir / "session.state.md").read_text()


def test_run_resolves_input_artifact_for_next_stage(prepare_module_case, run_cli):
    case_dir = prepare_module_case("artifact", "valid-two-stage-resolution")
    result = run_cli(
        ["run", "--config", str(case_dir / "octopus.yaml")],
        cwd=case_dir,
        env={
            "OPENOCTOPUS_DISABLE_ROLE_RUNTIME_LOOP": "0",
            "OPENOCTOPUS_DETERMINISTIC_RESULTS_AGENT_A": "SUCCESS",
            "OPENOCTOPUS_DETERMINISTIC_RESULTS_AGENT_B": "SUCCESS",
            "OPENOCTOPUS_DETERMINISTIC_OUTPUT_REFS_AGENT_A": str(case_dir / "sources" / "stage_a.md"),
            "OPENOCTOPUS_DETERMINISTIC_OUTPUT_REFS_AGENT_B": str(case_dir / "sources" / "stage_b.md"),
        },
    )

    assert result.returncode == 0, result.stderr
    session_dir = parse_session_dir(result.stdout)
    context_text = (session_dir / "roles" / "agent_b" / "context.md").read_text()
    assert "## input_artifacts" in context_text
    assert "ref: artifact_a" in context_text
    assert "content_ref: artifacts/artifact_a/0001/content.md" in context_text
    assert "manifest_ref: artifacts/artifact_a/0001/manifest.md" in context_text
    assert "status: COMPLETED" in (session_dir / "session.state.md").read_text()


def test_run_bumps_artifact_version(prepare_module_case, run_cli):
    case_dir = prepare_module_case("artifact", "valid-version-bump")
    result = run_cli(
        ["run", "--config", str(case_dir / "octopus.yaml")],
        cwd=case_dir,
        env={
            "OPENOCTOPUS_DISABLE_ROLE_RUNTIME_LOOP": "0",
            "OPENOCTOPUS_DETERMINISTIC_RESULTS_AGENT_A": "SUCCESS",
            "OPENOCTOPUS_DETERMINISTIC_RESULTS_AGENT_B": "SUCCESS",
            "OPENOCTOPUS_DETERMINISTIC_OUTPUT_REFS_AGENT_A": str(case_dir / "sources" / "v1.md"),
            "OPENOCTOPUS_DETERMINISTIC_OUTPUT_REFS_AGENT_B": str(case_dir / "sources" / "v2.md"),
        },
    )

    assert result.returncode == 0, result.stderr
    session_dir = parse_session_dir(result.stdout)
    artifact_dir = session_dir / "artifacts" / "artifact_a"
    assert (artifact_dir / "0001" / "manifest.md").exists()
    assert (artifact_dir / "0002" / "manifest.md").exists()
    diff_text = (artifact_dir / "0002" / "diff.md").read_text()
    assert "previous_hash:" in diff_text
    assert "current_hash:" in diff_text
    assert "version_count: 2" in (session_dir / "artifacts" / "index.md").read_text()


def test_run_executes_simple_codex_artifact_flow(prepare_module_case, run_cli):
    case_dir = prepare_module_case("artifact", "valid-codex-simple-flow")
    result = run_cli(
        ["run", "--config", str(case_dir / "octopus.yaml")],
        cwd=case_dir,
        env={"OPENOCTOPUS_DISABLE_ROLE_RUNTIME_LOOP": "0"},
    )

    assert result.returncode == 0, result.stderr
    session_dir = parse_session_dir(result.stdout)
    assert "status: COMPLETED" in (session_dir / "session.state.md").read_text()
    assert "executor_provider: codex" in (session_dir / "roles" / "agent_a" / "turns" / "0001-output.md").read_text()
    assert "executor_provider: codex" in (session_dir / "roles" / "agent_b" / "turns" / "0001-output.md").read_text()
    assert (session_dir / "artifacts" / "artifact_a" / "0001" / "content.md").exists()
    assert (session_dir / "artifacts" / "artifact_b" / "0001" / "content.md").exists()


def parse_session_dir(stdout: str) -> Path:
    prefix = "session created: "
    for line in stdout.splitlines():
        if line.startswith(prefix):
            return Path(line[len(prefix) :].strip())
    raise AssertionError(f"session path not found in stdout: {stdout}")
