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

from test.cli import run as cli_run, run_text as cli_text, wait_status, wait_a2s

TMP_DIR = '/tmp/syncloud'

requests.packages.urllib3.disable_warnings(InsecureRequestWarning)

@pytest.fixture(scope="session")
def module_setup(request, device, app_dir, artifact_dir):
    def module_teardown():
        device.run_ssh('ls -la /var/snap/games/current/config > {0}/config.ls.log'.format(TMP_DIR), throw=False)
        device.run_ssh('top -bn 1 -w 500 -c > {0}/top.log'.format(TMP_DIR), throw=False)
        device.run_ssh('ps auxfw > {0}/ps.log'.format(TMP_DIR), throw=False)
        device.run_ssh('netstat -nlp > {0}/netstat.log'.format(TMP_DIR), throw=False)
        device.run_ssh('journalctl | tail -2000 > {0}/journalctl.log'.format(TMP_DIR), throw=False)
        device.run_ssh('ls -la /snap > {0}/snap.ls.log'.format(TMP_DIR), throw=False)
        device.run_ssh('ls -la /snap/games > {0}/snap.ls.log'.format(TMP_DIR), throw=False)
        device.run_ssh('ls -la /var/snap/games > {0}/var.snap.ls.log'.format(TMP_DIR), throw=False)
        device.run_ssh('ls -la /var/snap/games/current/ > {0}/var.snap.current.ls.log'.format(TMP_DIR), throw=False)
        device.run_ssh('ls -la /var/snap/games/common > {0}/var.snap.common.ls.log'.format(TMP_DIR), throw=False)
        device.run_ssh('cat /etc/hosts > {0}/hosts.log'.format(TMP_DIR), throw=False)
        device.run_ssh('ls -la /snap/games/current/steamcmd > {0}/steamcmd.ls.log'.format(TMP_DIR), throw=False)
        device.run_ssh('ls /snap/games/current/steamcmd/lib32 | head -50 > {0}/steamcmd.lib32.log'.format(TMP_DIR), throw=False)
        device.run_ssh('cat /var/snap/games/current/.steam-home/Steam/logs/stderr.txt > {0}/steam.stderr.log 2>/dev/null'.format(TMP_DIR), throw=False)
        device.run_ssh('cat /var/snap/games/current/.steam-home/Steam/logs/bootstrap_log.txt > {0}/steam.bootstrap.log 2>/dev/null'.format(TMP_DIR), throw=False)
        device.run_ssh('ls -la /var/snap/games/current/.steam-home/Steam/logs/ > {0}/steam.logs.ls 2>/dev/null'.format(TMP_DIR), throw=False)

        device.scp_from_device('{0}/*'.format(TMP_DIR), artifact_dir)
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

def test_health(device):
    out = cli_text(device, 'health').strip()
    assert out == 'ok', out

def test_games_catalog(device):
    games = cli_run(device, 'games', 'list')
    ids = {g['id'] for g in games}
    assert 'teeworlds' in ids, 'teeworlds (CI install fixture) must be in catalog'
    assert 'cs2' in ids, 'cs2 (steam path) must be in catalog'
    assert 'hlds-cs' in ids, 'hlds-cs (CI Steam fixture) must be in catalog'
    assert 'cuberite' in ids, 'cuberite must be in catalog (disabled exemplar)'
    assert 'vanilla-bedrock' in ids, 'vanilla-bedrock must be in catalog (Android e2e target)'
    for g in games:
        assert g.get('tier') in ('supported', 'experimental', 'disabled'), \
            'game {} has invalid tier: {}'.format(g.get('id'), g)

def test_catalog_sources(device):
    sources = cli_run(device, 'games', 'sources')
    assert 'parkervcp' in sources
    assert 'linuxgsm' in sources

def test_servers_empty(device):
    assert cli_run(device, 'server', 'list') == []

def test_create_server(device):
    s = cli_run(device, 'server', 'create', 'test-tw', 'teeworlds', '--port', '8303')
    assert s['id'] > 0
    assert s['name'] == 'test-tw'
    assert s['gameId'] == 'teeworlds'
    assert s['status'] == 'stopped'

def test_list_after_create(device):
    servers = cli_run(device, 'server', 'list')
    assert len(servers) == 1
    assert servers[0]['name'] == 'test-tw'

def test_delete_server(device):
    cli_run(device, 'server', 'delete', 'test-tw')
    assert cli_run(device, 'server', 'list') == []

def test_create_unknown_game_rejected(device):
    out = device.run_ssh(
        '/snap/bin/games.cli server create bad not-a-game --port 1234 2>&1; echo EXIT=\$?',
        throw=False)
    assert 'unknown gameId' in out, out
    assert 'EXIT=1' in out, out

def test_logs_endpoint(device):
    cli_run(device, 'server', 'create', 'log-stub', 'teeworlds',
            '--port', '8304', '--start-cmd', 'echo hello-from-runner; sleep 5')
    cli_run(device, 'server', 'start', 'log-stub')
    time.sleep(2)
    lines = cli_run(device, 'server', 'logs', 'log-stub')
    assert any('hello-from-runner' in l for l in lines), 'log buffer should capture stdout: ' + str(lines)
    cli_run(device, 'server', 'stop', 'log-stub')
    cli_run(device, 'server', 'delete', 'log-stub')

def test_disabled_install_refused(device):
    """Disabled-tier game install must return a clean 409 with the
    disabledReason from the catalog entry, not a download attempt.
    Uses mohaa (URL 404, disabled) as the fixture since cuberite was
    promoted to experimental."""
    cli_run(device, 'server', 'create', 'disabled-stub', 'mohaa', '--port', '12203')
    out = device.run_ssh(
        '/snap/bin/games.cli server install disabled-stub 2>&1; echo EXIT=\$?',
        throw=False)
    assert 'disabled' in out.lower(), 'expected disabled message in: ' + out
    assert 'EXIT=1' in out, 'install command should exit non-zero: ' + out
    s = cli_run(device, 'server', 'show', 'disabled-stub')
    assert s['status'] == 'stopped', 'status must not flip to installing: ' + str(s)
    cli_run(device, 'server', 'delete', 'disabled-stub')

def test_cuberite_real_install_and_play(device):
    """Cuberite (~5MB C++ Minecraft Java-protocol server). Proves the
    bash-var URL substitution class of recipe and TCP 25565 bind.
    Cuberite spends ~20s generating world chunks before listening,
    so the bind check polls for up to 120s and uses the runner's log
    buffer as the source of truth (ss -tln from a separate SSH
    session sometimes misses snap-service IPv6 bindings)."""
    cli_run(device, 'server', 'create', 'cb-real', 'cuberite', '--port', '25565')
    cli_run(device, 'server', 'install', 'cb-real')
    wait_status(device, 'cb-real', 'stopped', timeout=180)

    install_dir = '/data/games/servers/cb-real'
    out = device.run_ssh('ls {0} 2>&1'.format(install_dir))
    assert 'Cuberite' in out, 'Cuberite binary missing post-install: ' + out

    s = cli_run(device, 'server', 'start', 'cb-real')
    assert s['status'] == 'running'

    deadline = time.time() + 120
    listening = False
    while time.time() < deadline:
        logs = cli_run(device, 'server', 'logs', 'cb-real') or []
        joined = '\n'.join(str(l) for l in logs)
        if 'Server Running On Port: 25565' in joined:
            listening = True
            break
        time.sleep(5)

    if not listening:
        logs = cli_run(device, 'server', 'logs', 'cb-real') or []
        print('cuberite stdout/stderr (last 50 lines):')
        for line in logs[-50:]:
            print(' ', line)
    assert listening, 'Cuberite should have logged "Server Running On Port: 25565" within 120s'

    cli_run(device, 'server', 'stop', 'cb-real')
    cli_run(device, 'server', 'delete', 'cb-real')

def test_vanilla_bedrock_real_install_and_play(device):
    """Mojang's official Bedrock dedicated server (~50MB zip). Proves
    the .zip extract path, amd64 wrap, and UDP 19132 bind that the
    Play Store Minecraft app on Android speaks. URL pinned in the
    catalog file; bump when Mojang ships a new version."""
    cli_run(device, 'server', 'create', 'bd-real', 'vanilla-bedrock', '--port', '19132')
    cli_run(device, 'server', 'install', 'bd-real')
    wait_status(device, 'bd-real', 'stopped', timeout=300)

    install_dir = '/data/games/servers/bd-real'
    out = device.run_ssh('ls {0} 2>&1'.format(install_dir))
    assert 'bedrock_server' in out, 'bedrock_server binary missing post-install: ' + out

    s = cli_run(device, 'server', 'start', 'bd-real')
    assert s['status'] == 'running'

    time.sleep(5)
    probe = device.run_ssh('ss -ulnp | grep 19132 || true')
    assert '19132' in probe, 'bedrock_server should be bound on udp:19132 — ss output: {0!r}'.format(probe)

    cli_run(device, 'server', 'stop', 'bd-real')
    cli_run(device, 'server', 'delete', 'bd-real')

def test_teeworlds_real_install_and_play(device):
    cli_run(device, 'server', 'create', 'tw-real', 'teeworlds', '--port', '8313')
    cli_run(device, 'server', 'install', 'tw-real')
    wait_status(device, 'tw-real', 'stopped', timeout=180)

    s = cli_run(device, 'server', 'start', 'tw-real')
    assert s['status'] == 'running'

    time.sleep(4)
    probe = device.run_ssh('ss -ulnp | grep 8313 || true')
    assert '8313' in probe, 'teeworlds_srv should be bound on udp:8313 — ss output: {0!r}'.format(probe)

    cli_run(device, 'server', 'stop', 'tw-real')
    cli_run(device, 'server', 'delete', 'tw-real')

@pytest.mark.flaky(retries=2, delay=15)
def test_hlds_cs_real_install(device):
    """Real CS 1.6 dedicated server install via SteamCMD (~822 MB download).

    Asserts the install path of phase 3b end-to-end: bundled SteamCMD,
    32-bit lib bundle (lib32/), runtime dir under $SNAP_DATA, +login
    anonymous +app_set_config 90 mod cstrike +app_update 90 validate.

    Starting hlds_linux is xfail'd separately — it requires a Steam
    Auth Server reachable for SteamAPI_Init / IClientUtils, which the
    snap can't provide without a full Steam runtime emulator."""
    for s in cli_run(device, 'server', 'list') or []:
        if s.get('name') == 'hlds-real':
            cli_run(device, 'server', 'delete', 'hlds-real')
    device.run_ssh('rm -rf /data/games/servers/hlds-real', throw=False)

    cli_run(device, 'server', 'create', 'hlds-real', 'hlds-cs', '--port', '27115')
    cli_run(device, 'server', 'install', 'hlds-real')
    wait_status(device, 'hlds-real', 'stopped', timeout=900)

    out = device.run_ssh('ls /data/games/servers/hlds-real/hlds_linux 2>&1')
    assert 'hlds_linux' in out and 'No such file' not in out, \
        'hlds_linux missing post-install: ' + out

    cli_run(device, 'server', 'delete', 'hlds-real')

@pytest.mark.xfail(
    reason='HLDS startup needs Steam Pipe / lsteamclient shim to satisfy '
           'SteamAPI_Init -> IClientUtils::GetConnectedUniverse. SteamCMD '
           'install itself works (see test_hlds_cs_real_install).',
    strict=False, run=True)
def test_hlds_cs_a2s_query(device):
    cli_run(device, 'server', 'create', 'hlds-query', 'hlds-cs', '--port', '27116')
    cli_run(device, 'server', 'install', 'hlds-query')
    wait_status(device, 'hlds-query', 'stopped', timeout=900)
    cli_run(device, 'server', 'start', 'hlds-query')

    info = wait_a2s(device, 'hlds-query', timeout=60)
    assert 'Counter-Strike' in info.get('Game', '') or 'cstrike' in info.get('Folder', ''), info

    cli_run(device, 'server', 'stop', 'hlds-query')
    cli_run(device, 'server', 'delete', 'hlds-query')

def test_lifecycle(device):
    cli_run(device, 'server', 'create', 'stub', 'teeworlds',
            '--port', '8303', '--start-cmd', 'sleep 30')

    s = cli_run(device, 'server', 'start', 'stub')
    assert s['status'] == 'running'

    again = device.run_ssh('/snap/bin/games.cli server start stub 2>&1; echo EXIT=\$?', throw=False)
    assert 'already running' in again, again
    assert 'EXIT=1' in again, again

    s = cli_run(device, 'server', 'stop', 'stub')
    assert s['status'] == 'stopped'

    cli_run(device, 'server', 'delete', 'stub')

def test_storage_change_event(device):
    device.run_ssh('snap run games.storage-change > {0}/storage-change.log'.format(TMP_DIR))

def test_access_change_event(device):
    device.run_ssh('snap run games.access-change > {0}/access-change.log'.format(TMP_DIR))

def test_remove(device, app):
    response = device.app_remove(app)
    assert response.status_code == 200, response.text

def test_reinstall(app_archive_path, device_host, device_password):
    local_install(device_host, device_password, app_archive_path)
