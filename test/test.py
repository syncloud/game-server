import os
from os.path import join
from subprocess import check_output

import pytest
import requests
from requests.packages.urllib3.exceptions import InsecureRequestWarning
from syncloudlib.http import wait_for_rest
from syncloudlib.integration.hosts import add_host_alias
from syncloudlib.integration.installer import local_install

TMP_DIR = '/tmp/syncloud'

requests.packages.urllib3.disable_warnings(InsecureRequestWarning)


@pytest.fixture(scope="session")
def module_setup(request, device, app_dir, artifact_dir):
    def module_teardown():
        device.run_ssh('ls -la /var/snap/game-server/current/config > {0}/config.ls.log'.format(TMP_DIR), throw=False)
        device.run_ssh('top -bn 1 -w 500 -c > {0}/top.log'.format(TMP_DIR), throw=False)
        device.run_ssh('ps auxfw > {0}/ps.log'.format(TMP_DIR), throw=False)
        device.run_ssh('netstat -nlp > {0}/netstat.log'.format(TMP_DIR), throw=False)
        device.run_ssh('journalctl | tail -2000 > {0}/journalctl.log'.format(TMP_DIR), throw=False)
        device.run_ssh('ls -la /snap > {0}/snap.ls.log'.format(TMP_DIR), throw=False)
        device.run_ssh('ls -la /snap/game-server > {0}/snap.ls.log'.format(TMP_DIR), throw=False)
        device.run_ssh('ls -la /var/snap/game-server > {0}/var.snap.ls.log'.format(TMP_DIR), throw=False)
        device.run_ssh('ls -la /var/snap/game-server/current/ > {0}/var.snap.current.ls.log'.format(TMP_DIR), throw=False)
        device.run_ssh('ls -la /var/snap/game-server/common > {0}/var.snap.common.ls.log'.format(TMP_DIR), throw=False)
        device.run_ssh('cat /etc/hosts > {0}/hosts.log'.format(TMP_DIR), throw=False)

        app_log_dir = join(artifact_dir, 'log')
        os.mkdir(app_log_dir)
        device.scp_from_device('{0}/*'.format(TMP_DIR), app_log_dir)
        check_output('chmod -R a+r {0}'.format(artifact_dir), shell=True)

    request.addfinalizer(module_teardown)


def test_start(module_setup, device, device_host, app, domain):
    add_host_alias(app, device_host, domain)
    device.run_ssh('date', retries=100)
    device.run_ssh('mkdir {0}'.format(TMP_DIR))


@pytest.mark.flaky(retries=50, delay=10)
def test_activate_device(device):
    response = device.activate_custom()
    assert response.status_code == 200, response.text


def test_install(app_archive_path, device_host, device_password, device):
    local_install(device_host, device_password, app_archive_path)


def test_index(app_domain):
    wait_for_rest(requests.session(), "https://{0}".format(app_domain), 200, 10)


def test_health(app_domain):
    response = requests.get('https://{0}/api/v1/health'.format(app_domain), verify=False)
    assert response.status_code == 200, response.text
    assert response.json().get('status') == 'ok', response.text


def test_games_catalog(app_domain):
    response = requests.get('https://{0}/api/v1/games'.format(app_domain), verify=False)
    assert response.status_code == 200, response.text
    games = response.json()
    ids = {g['id'] for g in games}
    assert 'teeworlds' in ids, 'teeworlds (smallest pelican egg, our CI fixture) must be in catalog'
    assert 'cs2' in ids, 'cs2 anonymous-friendly steam server must be in catalog'


def test_servers_empty(app_domain):
    response = requests.get('https://{0}/api/v1/servers'.format(app_domain), verify=False)
    assert response.status_code == 200, response.text
    assert response.json() == [], 'no servers installed at start'


def test_create_server(app_domain):
    response = requests.post(
        'https://{0}/api/v1/servers'.format(app_domain),
        json={'name': 'test-tw', 'gameId': 'teeworlds', 'port': 8303},
        verify=False)
    assert response.status_code == 201, response.text
    body = response.json()
    assert body['id'] > 0
    assert body['name'] == 'test-tw'
    assert body['gameId'] == 'teeworlds'
    assert body['status'] == 'stopped'


def test_list_after_create(app_domain):
    response = requests.get('https://{0}/api/v1/servers'.format(app_domain), verify=False)
    assert response.status_code == 200
    servers = response.json()
    assert len(servers) == 1
    assert servers[0]['name'] == 'test-tw'


def test_delete_server(app_domain):
    list_resp = requests.get('https://{0}/api/v1/servers'.format(app_domain), verify=False)
    sid = list_resp.json()[0]['id']
    d = requests.delete('https://{0}/api/v1/servers/{1}'.format(app_domain, sid), verify=False)
    assert d.status_code == 204, d.text
    after = requests.get('https://{0}/api/v1/servers'.format(app_domain), verify=False).json()
    assert after == []


def test_create_unknown_game_rejected(app_domain):
    r = requests.post(
        'https://{0}/api/v1/servers'.format(app_domain),
        json={'name': 'bad', 'gameId': 'not-a-game', 'port': 1234},
        verify=False)
    assert r.status_code == 400, r.text


def test_logs_endpoint(app_domain):
    create = requests.post(
        'https://{0}/api/v1/servers'.format(app_domain),
        json={
            'name': 'log-stub',
            'gameId': 'teeworlds',
            'port': 8304,
            'startCmd': 'echo hello-from-runner; sleep 5',
        },
        verify=False)
    assert create.status_code == 201, create.text
    sid = create.json()['id']
    start = requests.post(
        'https://{0}/api/v1/servers/{1}/start'.format(app_domain, sid),
        verify=False)
    assert start.status_code == 200, start.text
    import time
    time.sleep(2)
    logs = requests.get(
        'https://{0}/api/v1/servers/{1}/logs'.format(app_domain, sid),
        verify=False)
    assert logs.status_code == 200, logs.text
    lines = logs.json().get('lines', [])
    assert any('hello-from-runner' in l for l in lines), 'log buffer should capture stdout: ' + str(lines)
    requests.post('https://{0}/api/v1/servers/{1}/stop'.format(app_domain, sid), verify=False)
    requests.delete('https://{0}/api/v1/servers/{1}'.format(app_domain, sid), verify=False)


def test_lifecycle(app_domain):
    create = requests.post(
        'https://{0}/api/v1/servers'.format(app_domain),
        json={
            'name': 'stub',
            'gameId': 'teeworlds',
            'port': 8303,
            'startCmd': 'sleep 30',
        },
        verify=False)
    assert create.status_code == 201, create.text
    sid = create.json()['id']

    start = requests.post(
        'https://{0}/api/v1/servers/{1}/start'.format(app_domain, sid),
        verify=False)
    assert start.status_code == 200, start.text
    assert start.json()['status'] == 'running'

    again = requests.post(
        'https://{0}/api/v1/servers/{1}/start'.format(app_domain, sid),
        verify=False)
    assert again.status_code == 409, again.text

    stop = requests.post(
        'https://{0}/api/v1/servers/{1}/stop'.format(app_domain, sid),
        verify=False)
    assert stop.status_code == 200, stop.text
    assert stop.json()['status'] == 'stopped'

    cleanup = requests.delete(
        'https://{0}/api/v1/servers/{1}'.format(app_domain, sid),
        verify=False)
    assert cleanup.status_code == 204, cleanup.text


def test_storage_change_event(device):
    device.run_ssh('snap run game-server.storage-change > {0}/storage-change.log'.format(TMP_DIR))


def test_access_change_event(device):
    device.run_ssh('snap run game-server.access-change > {0}/access-change.log'.format(TMP_DIR))


def test_remove(device, app):
    response = device.app_remove(app)
    assert response.status_code == 200, response.text


def test_reinstall(app_archive_path, device_host, device_password):
    local_install(device_host, device_password, app_archive_path)
