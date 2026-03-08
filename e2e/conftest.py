import os
import shutil
import subprocess
from pathlib import Path
from typing import Optional

import pytest


@pytest.fixture(scope="session")
def project_root() -> Path:
    return Path(__file__).resolve().parent.parent


@pytest.fixture(scope="session")
def codex_home() -> Path:
    return Path.home() / ".codex"


@pytest.fixture(scope="session")
def real_codex_env(codex_home: Path) -> Path:
    auth_file = codex_home / "auth.json"
    config_file = codex_home / "config.toml"
    assert auth_file.exists(), f"missing Codex auth file: {auth_file}"
    assert config_file.exists(), f"missing Codex config file: {config_file}"
    result = subprocess.run(["codex", "--version"], capture_output=True, text=True)
    assert result.returncode == 0, result.stderr or result.stdout
    return codex_home


@pytest.fixture(scope="session")
def workspace_dir(project_root: Path) -> Path:
    workspace = project_root / "e2e-test"
    if workspace.exists():
        shutil.rmtree(workspace)
    workspace.mkdir(parents=True, exist_ok=True)
    return workspace


@pytest.fixture(scope="session")
def binary_path(project_root: Path, workspace_dir: Path) -> Path:
    binary = workspace_dir / "bin" / "openoctopus"
    binary.parent.mkdir(parents=True, exist_ok=True)
    subprocess.run(
        ["go", "build", "-o", str(binary), "./cmd/openoctopus"],
        cwd=project_root,
        check=True,
    )
    return binary


@pytest.fixture()
def prepare_case(workspace_dir: Path, request):
    fixtures_root = Path(__file__).resolve().parent / "config" / "fixtures"

    def _prepare(case_name: str) -> Path:
        source = fixtures_root / case_name
        target = workspace_dir / case_name
        if target.exists():
            shutil.rmtree(target)
        shutil.copytree(source, target)
        return target

    return _prepare


@pytest.fixture()
def run_cli(binary_path: Path, real_codex_env: Path):
    def _run(args, cwd: Path, env: Optional[dict] = None):
        runtime_env = os.environ.copy()
        runtime_env["CODEX_HOME"] = str(real_codex_env)
        if env:
            runtime_env.update(env)
        return subprocess.run(
            [str(binary_path), *args],
            cwd=cwd,
            env=runtime_env,
            capture_output=True,
            text=True,
        )

    return _run
