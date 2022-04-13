import os.path
from pathlib import Path
import shutil

import pytest

from core.compose_factory import create_docker_compose
import core.wrappers as wrappers
from core.utils import setup_logger


logger = setup_logger(__name__)


@pytest.fixture
def server_service():
    service_name = "server"
    compose = create_docker_compose(service_name)
    compose.start()
    compose.wait_for_healthy(service_name)
    wrapper = wrappers.Server(compose, service_name)
    return wrapper


@pytest.fixture
def kea_service(request):
    param = {
        "service_name": "agent-kea",
        "suppress_registration": False
    }

    if request.param is not None:
        param.update(request.param)

    env_vars = None
    if param['suppress_registration']:
        env_vars = { "STORK_SERVER_URL": "" }

    service_name = param['service_name']
    compose = create_docker_compose(service_name, env_vars)
    compose.start()
    compose.wait_for_healthy(service_name)
    wrapper = wrappers.Kea(compose, service_name)
    return wrapper


@pytest.fixture(autouse=True)
def finish(request):
    """Save all logs to file and down all used containers."""
    function_name = request.function.__name__
    def collect_logs_and_down_all():
        logger.info('COLLECTING LOGS')

        # Collect logs
        compose = create_docker_compose()
        stdout, stderr = compose.get_logs()

        # prepare test directory for logs, etc
        tests_dir = Path('test-results')
        tests_dir.mkdir(exist_ok=True)
        test_name = function_name
        test_name = test_name.replace('[', '__')
        test_name = test_name.replace('/', '_')
        test_name = test_name.replace(']', '')
        test_dir = tests_dir / test_name
        if test_dir.exists():
            shutil.rmtree(test_dir)
        test_dir.mkdir()

        # Write logs
        with open(test_dir / "stdout.log", 'wt') as f:
            f.write(stdout)

        with open(test_dir / "stderr.log", 'wt') as f:
            f.write(stderr)

        # Stop all containers
        compose.stop()
    request.addfinalizer(collect_logs_and_down_all)