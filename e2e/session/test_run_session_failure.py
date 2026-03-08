def test_run_rolls_back_when_sessions_dir_path_is_occupied(prepare_module_case, run_cli):
    case_dir = prepare_module_case("session", "valid-path-collision")
    occupied_path = case_dir / "session-target"
    occupied_path.write_text("occupied")

    result = run_cli(["run", "--config", str(case_dir / "octopus.yaml")], cwd=case_dir)

    assert result.returncode != 0
    assert occupied_path.is_file()
    assert list(case_dir.glob("sess_*")) == []
