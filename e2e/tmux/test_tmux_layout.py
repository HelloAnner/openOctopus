import json
import subprocess
from pathlib import Path


def test_run_bootstraps_tmux_layout_and_capture(prepare_module_case, run_cli):
    case_dir = prepare_module_case("tmux", "valid-basic-layout")

    result = run_cli(["run", "--config", str(case_dir / "octopus.yaml")], cwd=case_dir)
    assert result.returncode == 0, result.stderr

    session_dir = parse_session_dir(result.stdout)
    layout_path = session_dir / "state" / "tmux" / "layout.md"
    assert layout_path.exists()

    layout = read_layout(layout_path)
    assert layout["socket_name"]
    assert layout["session_name"]
    assert "role_id: agent_a" in layout_path.read_text()
    assert "role_id: agent_b" in layout_path.read_text()

    try:
        assert tmux_has_session(layout["socket_name"], layout["session_name"])
        pane_map = list_panes(layout["socket_name"], layout["session_name"])
        assert "main" in pane_map
        assert "role:agent_a" in pane_map
        assert "role:agent_b" in pane_map

        main_output = capture_pane(layout["socket_name"], pane_map["main"])
        role_output = capture_pane(layout["socket_name"], pane_map["role:agent_a"])
        assert "[openoctopus] main session=" in main_output
        assert "[openoctopus] role=agent_a" in role_output
    finally:
        cleanup_tmux(layout["socket_name"], layout["session_name"])


def test_switch_returns_target_json_without_attach(prepare_module_case, run_cli):
    case_dir = prepare_module_case("tmux", "valid-basic-layout")

    run_result = run_cli(["run", "--config", str(case_dir / "octopus.yaml")], cwd=case_dir)
    assert run_result.returncode == 0, run_result.stderr
    session_dir = parse_session_dir(run_result.stdout)
    layout = read_layout(session_dir / "state" / "tmux" / "layout.md")

    try:
        switch_result = run_cli(
            ["switch", "--session", str(session_dir), "--role", "agent_b", "--format", "json"],
            cwd=case_dir,
        )
        assert switch_result.returncode == 0, switch_result.stderr
        payload = json.loads(switch_result.stdout)
        assert payload["ok"] is True
        assert payload["data"]["session_dir"] == str(session_dir)
        assert payload["data"]["socket_name"] == layout["socket_name"]
        assert payload["data"]["session_name"] == layout["session_name"]
        assert payload["data"]["target_role"] == "agent_b"
        assert payload["data"]["target_pane_id"]
        assert payload["data"]["switched"] is False
    finally:
        cleanup_tmux(layout["socket_name"], layout["session_name"])


def test_validate_rejects_invalid_tmux_layout(prepare_module_case, run_cli):
    case_dir = prepare_module_case("tmux", "invalid-layout-config")

    result = run_cli(["validate", "--config", str(case_dir / "octopus.yaml")], cwd=case_dir)

    assert result.returncode != 0
    assert "runtime.tmux.main_pane_ratio" in result.stderr
    assert "runtime.tmux.role_layout" in result.stderr


def parse_session_dir(stdout: str) -> Path:
    prefix = "session created: "
    for line in stdout.splitlines():
        if line.startswith(prefix):
            return Path(line[len(prefix) :].strip())
    raise AssertionError(f"session path not found in stdout: {stdout}")


def read_layout(path: Path) -> dict:
    values = {}
    for line in path.read_text().splitlines():
        stripped = line.strip()
        if not stripped.startswith("- "):
            continue
        key, _, value = stripped[2:].partition(":")
        values[key.strip()] = value.strip()
    return values


def list_panes(socket_name: str, session_name: str) -> dict:
    result = subprocess.run(
        ["tmux", "-L", socket_name, "list-panes", "-t", session_name, "-F", "#{pane_id}|#{pane_title}"],
        capture_output=True,
        text=True,
        check=True,
    )
    pane_map = {}
    for line in result.stdout.splitlines():
        pane_id, _, title = line.partition("|")
        pane_map[title.strip()] = pane_id.strip()
    return pane_map


def capture_pane(socket_name: str, pane_id: str) -> str:
    result = subprocess.run(
        ["tmux", "-L", socket_name, "capture-pane", "-p", "-t", pane_id],
        capture_output=True,
        text=True,
        check=True,
    )
    return result.stdout


def tmux_has_session(socket_name: str, session_name: str) -> bool:
    result = subprocess.run(
        ["tmux", "-L", socket_name, "has-session", "-t", session_name],
        capture_output=True,
        text=True,
    )
    return result.returncode == 0


def cleanup_tmux(socket_name: str, session_name: str):
    subprocess.run(
        ["tmux", "-L", socket_name, "kill-session", "-t", session_name],
        capture_output=True,
        text=True,
        check=False,
    )
