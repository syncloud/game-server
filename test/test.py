import os
import time
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
def auth(device_user, device_password):
    return (device_user, device_password)


@pytest.fixture(scope="session")
def api(app_domain):
    return 'https://{0}/api/v1'.format(app_domain)


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
        device.run_ssh('ls -la /snap/game-server/current/steamcmd > {0}/steamcmd.ls.log'.format(TMP_DIR), throw=False)
        device.run_ssh('ls /snap/game-server/current/steamcmd/lib32 | head -50 > {0}/steamcmd.lib32.log'.format(TMP_DIR), throw=False)
        device.run_ssh('cat /var/snap/game-server/current/.steam-home/Steam/logs/stderr.txt > {0}/steam.stderr.log 2>/dev/null'.format(TMP_DIR), throw=False)
        device.run_ssh('cat /var/snap/game-server/current/.steam-home/Steam/logs/bootstrap_log.txt > {0}/steam.bootstrap.log 2>/dev/null'.format(TMP_DIR), throw=False)
        device.run_ssh('ls -la /var/snap/game-server/current/.steam-home/Steam/logs/ > {0}/steam.logs.ls 2>/dev/null'.format(TMP_DIR), throw=False)

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


def test_health(api, auth):
    response = requests.get(api + '/health', auth=auth, verify=False)
    assert response.status_code == 200, response.text
    assert response.json().get('status') == 'ok', response.text


def test_games_catalog(api, auth):
    response = requests.get(api + '/games', auth=auth, verify=False)
    assert response.status_code == 200, response.text
    games = response.json()
    ids = {g['id'] for g in games}
    assert 'teeworlds' in ids, 'teeworlds (smallest pelican egg, our CI fixture) must be in catalog'
    assert 'cs2' in ids, 'cs2 anonymous-friendly steam server must be in catalog'


def test_servers_empty(api, auth):
    response = requests.get(api + '/servers', auth=auth, verify=False)
    assert response.status_code == 200, response.text
    assert response.json() == [], 'no servers installed at start'


def test_create_server(api, auth):
    response = requests.post(
        api + '/servers',
        auth=auth,
        json={'name': 'test-tw', 'gameId': 'teeworlds', 'port': 8303},
        verify=False)
    assert response.status_code == 201, response.text
    body = response.json()
    assert body['id'] > 0
    assert body['name'] == 'test-tw'
    assert body['gameId'] == 'teeworlds'
    assert body['status'] == 'stopped'


def test_list_after_create(api, auth):
    response = requests.get(api + '/servers', auth=auth, verify=False)
    assert response.status_code == 200
    servers = response.json()
    assert len(servers) == 1
    assert servers[0]['name'] == 'test-tw'


def test_delete_server(api, auth):
    list_resp = requests.get(api + '/servers', auth=auth, verify=False)
    sid = list_resp.json()[0]['id']
    d = requests.delete(api + '/servers/{0}'.format(sid), auth=auth, verify=False)
    assert d.status_code == 204, d.text
    after = requests.get(api + '/servers', auth=auth, verify=False).json()
    assert after == []


def test_create_unknown_game_rejected(api, auth):
    r = requests.post(
        api + '/servers',
        auth=auth,
        json={'name': 'bad', 'gameId': 'not-a-game', 'port': 1234},
        verify=False)
    assert r.status_code == 400, r.text


def test_logs_endpoint(api, auth):
    create = requests.post(
        api + '/servers',
        auth=auth,
        json={
            'name': 'log-stub',
            'gameId': 'teeworlds',
            'port': 8304,
            'startCmd': 'echo hello-from-runner; sleep 5',
        },
        verify=False)
    assert create.status_code == 201, create.text
    sid = create.json()['id']
    start = requests.post(api + '/servers/{0}/start'.format(sid), auth=auth, verify=False)
    assert start.status_code == 200, start.text
    time.sleep(2)
    logs = requests.get(api + '/servers/{0}/logs'.format(sid), auth=auth, verify=False)
    assert logs.status_code == 200, logs.text
    lines = logs.json().get('lines', [])
    assert any('hello-from-runner' in l for l in lines), 'log buffer should capture stdout: ' + str(lines)
    requests.post(api + '/servers/{0}/stop'.format(sid), auth=auth, verify=False)
    requests.delete(api + '/servers/{0}'.format(sid), auth=auth, verify=False)


def _wait_a2s(api, auth, sid, timeout):
    deadline = time.time() + timeout
    last = None
    while time.time() < deadline:
        r = requests.get(api + '/servers/{0}/query'.format(sid), auth=auth, verify=False)
        last = (r.status_code, r.text[:200])
        if r.status_code == 200:
            return r.json()
        time.sleep(3)
    raise AssertionError('timeout waiting for A2S response, last={0}'.format(last))


def _wait_status(api, auth, sid, target, timeout):
    deadline = time.time() + timeout
    last = None
    while time.time() < deadline:
        r = requests.get(api + '/servers/{0}'.format(sid), auth=auth, verify=False)
        if r.status_code == 200:
            last = r.json().get('status')
            if last == target:
                return last
            if last == 'install-error':
                raise AssertionError('install errored: ' + r.text)
        time.sleep(2)
    raise AssertionError('timeout waiting for status={0}, last={1}'.format(target, last))


def test_teeworlds_real_install_and_play(api, auth, device):
    create = requests.post(
        api + '/servers',
        auth=auth,
        json={'name': 'tw-real', 'gameId': 'teeworlds', 'port': 8313},
        verify=False)
    assert create.status_code == 201, create.text
    sid = create.json()['id']

    install = requests.post(api + '/servers/{0}/install'.format(sid), auth=auth, verify=False)
    assert install.status_code == 200, install.text
    _wait_status(api, auth, sid, 'stopped', timeout=180)

    start = requests.post(api + '/servers/{0}/start'.format(sid), auth=auth, verify=False)
    assert start.status_code == 200, start.text
    assert start.json()['status'] == 'running'

    time.sleep(4)
    probe = device.run_ssh('ss -ulnp | grep 8313 || true')
    assert '8313' in probe, 'teeworlds_srv should be bound on udp:8313 — ss output: {0!r}'.format(probe)

    stop = requests.post(api + '/servers/{0}/stop'.format(sid), auth=auth, verify=False)
    assert stop.status_code == 200, stop.text

    cleanup = requests.delete(api + '/servers/{0}'.format(sid), auth=auth, verify=False)
    assert cleanup.status_code == 204, cleanup.text


def test_steamcmd_diagnostics(device):
    """Capture exactly what steamcmd needs and what's missing. Always runs;
    output goes to pytest stdout and is visible in the CI log. Not an
    assertion — diagnostic only."""

    def show(title, cmd):
        print('\n===== {} =====\nCMD: {}'.format(title, cmd))
        out = device.run_ssh(cmd, throw=False)
        print(out if out else '(no output)')

    # heredoc body — use simple bash, no parens in echo args
    diag = '''#!/bin/bash
set +e
echo SECTION df
df -h /var/snap /tmp / 2>&1
echo SECTION statf-varsnap
stat -f /var/snap/game-server/current 2>&1
echo SECTION statf-tmp
stat -f /tmp 2>&1
echo SECTION apt-strace
apt-get install -y strace 2>&1 | tail -3
echo SECTION seed-runtime
/snap/game-server/current/bin/steamcmd.sh +exit 2>&1 | head -20
echo SECTION ls-runtime
ls -la /var/snap/game-server/current/.steam-runtime/ 2>&1
echo SECTION steamlogs-after-seed
ls -la /var/snap/game-server/current/.steam-home/Steam/logs/ 2>&1
cat /var/snap/game-server/current/.steam-home/Steam/logs/stderr.txt 2>&1 | head -20
echo SECTION exp1-run-from-tmp
rm -rf /tmp/sctest
mkdir -p /tmp/sctest
cp -r /snap/game-server/current/steamcmd/linux32 /tmp/sctest/
cd /tmp/sctest
HOME=/tmp/sctest LD_LIBRARY_PATH=/snap/game-server/current/steamcmd/lib32 /snap/game-server/current/steamcmd/lib32/ld-linux.so.2 --library-path /tmp/sctest/linux32:/snap/game-server/current/steamcmd/lib32 /tmp/sctest/linux32/steamcmd +exit 2>&1 | head -20
echo SECTION strace-bare
cd /var/snap/game-server/current/.steam-runtime
strace -f -e trace=openat,connect,statfs,fstatfs,access,write -ttt -s 200 -o /tmp/strace.log /snap/game-server/current/bin/steamcmd.sh +exit 2>&1 | head -20
echo SECTION strace-tail
tail -150 /tmp/strace.log 2>&1
echo SECTION done
'''
    device.run_ssh("cat > /tmp/diag.sh <<'DIAGEOF'\n" + diag + "DIAGEOF\n", throw=False)
    show('comprehensive diag', 'bash /tmp/diag.sh')

    show('ldd on the 32-bit steamcmd binary (look for "not found")',
         'ldd /snap/game-server/current/steamcmd/linux32/steamcmd 2>&1 || true')

    show('ldd via our bundled loader',
         'LD_LIBRARY_PATH=/snap/game-server/current/steamcmd/lib32 '
         '/snap/game-server/current/steamcmd/lib32/ld-linux.so.2 --verify '
         '/snap/game-server/current/steamcmd/linux32/steamcmd 2>&1 || true')

    show('lib32 file listing',
         'ls /snap/game-server/current/steamcmd/lib32 | sort')

    show('lib32 contains key SSL/curl/nss libs?',
         'ls /snap/game-server/current/steamcmd/lib32 | grep -E '
         '"libssl|libcurl|libnghttp2|libidn2|libnss|libgssapi|libkrb5|'
         'libsasl|libssh|librtmp|libpsl|libcom_err|libkeyutils|libbrotli" '
         '| sort')

    show('host can resolve Steam CDN',
         'getent hosts steamcdn-a.akamaihd.net 2>&1; '
         'getent hosts client-update.akamaihd.net 2>&1')

    show('host TLS works to Steam',
         'curl -sIv --max-time 10 https://steamcdn-a.akamaihd.net/client/ 2>&1 '
         '| head -40 || true')

    show('runtime dir state after a bare steamcmd +exit',
         'sudo -u game-server -E HOME=/var/snap/game-server/current/.steam-home '
         '/snap/game-server/current/bin/steamcmd.sh +exit 2>&1 | head -40 || true')

    show('Steam logs after that attempt',
         'cat /var/snap/game-server/current/.steam-home/Steam/logs/stderr.txt 2>&1 || true; '
         'echo ---bootstrap---; '
         'cat /var/snap/game-server/current/.steam-home/Steam/logs/bootstrap_log.txt 2>&1 || true')

    show('LD_DEBUG=libs on a +exit (first 80 lines)',
         'sudo -u game-server -E HOME=/var/snap/game-server/current/.steam-home '
         'LD_DEBUG=libs LD_DEBUG_OUTPUT=/tmp/ld-debug '
         '/snap/game-server/current/bin/steamcmd.sh +exit 2>&1 >/dev/null || true; '
         'cat /tmp/ld-debug.* 2>/dev/null | head -200 || echo "(no ld-debug output)"')


def test_hlds_cs_real_install_and_query(api, auth, device):
    create = requests.post(
        api + '/servers',
        auth=auth,
        json={'name': 'hlds-real', 'gameId': 'hlds-cs', 'port': 27115},
        verify=False)
    assert create.status_code == 201, create.text
    sid = create.json()['id']

    install = requests.post(api + '/servers/{0}/install'.format(sid), auth=auth, verify=False)
    assert install.status_code == 200, install.text
    _wait_status(api, auth, sid, 'stopped', timeout=900)

    start = requests.post(api + '/servers/{0}/start'.format(sid), auth=auth, verify=False)
    assert start.status_code == 200, start.text

    info = _wait_a2s(api, auth, sid, timeout=60)
    assert 'Counter-Strike' in info.get('Game', '') or 'cstrike' in info.get('Folder', ''), info

    requests.post(api + '/servers/{0}/stop'.format(sid), auth=auth, verify=False)
    requests.delete(api + '/servers/{0}'.format(sid), auth=auth, verify=False)


def test_lifecycle(api, auth):
    create = requests.post(
        api + '/servers',
        auth=auth,
        json={
            'name': 'stub',
            'gameId': 'teeworlds',
            'port': 8303,
            'startCmd': 'sleep 30',
        },
        verify=False)
    assert create.status_code == 201, create.text
    sid = create.json()['id']

    start = requests.post(api + '/servers/{0}/start'.format(sid), auth=auth, verify=False)
    assert start.status_code == 200, start.text
    assert start.json()['status'] == 'running'

    again = requests.post(api + '/servers/{0}/start'.format(sid), auth=auth, verify=False)
    assert again.status_code == 409, again.text

    stop = requests.post(api + '/servers/{0}/stop'.format(sid), auth=auth, verify=False)
    assert stop.status_code == 200, stop.text
    assert stop.json()['status'] == 'stopped'

    cleanup = requests.delete(api + '/servers/{0}'.format(sid), auth=auth, verify=False)
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
