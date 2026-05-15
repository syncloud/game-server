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

def test_health(device):
    out = cli_text(device, 'health').strip()
    assert out == 'ok', out

def test_games_catalog(device):
    games = cli_run(device, 'games', 'list')
    ids = {g['id'] for g in games}
    assert 'teeworlds' in ids, 'teeworlds (smallest pelican egg, our CI fixture) must be in catalog'
    assert 'cs2' in ids, 'cs2 anonymous-friendly steam server must be in catalog'
    assert 'hlds-cs' in ids, 'hlds-cs (CI Steam fixture) must be in catalog'
    for g in games:
        assert g.get('tier') in ('verified', 'compatible', 'experimental'), \
            'game {} has no tier: {}'.format(g.get('id'), g)

def test_catalog_sources(device):
    sources = cli_run(device, 'games', 'sources')
    assert 'parkervcp/eggs' in sources
    assert 'pelican-eggs/games' in sources

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
        'games.cli server create bad not-a-game --port 1234 2>&1; echo EXIT=$?',
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

def test_steamcmd_diagnostics(device):
    def show(title, cmd):
        print('\n===== {} =====\nCMD: {}'.format(title, cmd))
        out = device.run_ssh(cmd, throw=False)
        print(out if out else '(no output)')

    diag = '''#!/bin/bash
set +e
echo SECTION df
df -h /var/snap /tmp / 2>&1
echo SECTION statf-varsnap
stat -f /var/snap/games/current 2>&1
echo SECTION statf-tmp
stat -f /tmp 2>&1
echo SECTION apt-strace
apt-get install -y strace 2>&1 | tail -3
echo SECTION 32bit-curl-test
ls -la /snap/games/current/steamcmd/lib32/curl.i386 2>&1
echo --- 32-bit curl HEAD to Steam CDN ---
LD_LIBRARY_PATH=/snap/games/current/steamcmd/lib32 /snap/games/current/steamcmd/lib32/ld-linux.so.2 --library-path /snap/games/current/steamcmd/lib32 /snap/games/current/steamcmd/lib32/curl.i386 -sIv --max-time 10 https://steamcdn-a.akamaihd.net/client/ 2>&1 | head -40
echo SECTION seed-runtime-as-games
sudo -u games -H bash -c "/snap/games/current/bin/steamcmd.sh +exit" 2>&1 | head -20
echo SECTION ls-runtime
ls -la /var/snap/games/current/.steam-runtime/ 2>&1
echo SECTION steamlogs-after-seed
ls -la /var/snap/games/current/.steam-home/Steam/logs/ 2>&1
cat /var/snap/games/current/.steam-home/Steam/logs/stderr.txt 2>&1 | head -20
echo SECTION done
'''
    device.run_ssh("cat > /tmp/diag.sh <<'DIAGEOF'\n" + diag + "DIAGEOF\n", throw=False)
    show('comprehensive diag', 'bash /tmp/diag.sh')

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
    reason="Paper/Fabric/etc. Minecraft eggs need `jq` (Paper) or apt "
           "(Fabric, Glowstone) at install time; neither is available "
           "in the snap install context. Tracked as follow-up: vendor "
           "a curated minecraft-vanilla entry that fetches server.jar "
           "from Mojang launchermeta with curl alone, no jq.",
    strict=False, run=True)
def test_minecraft_real_install_and_play(device):
    """Full Minecraft cycle: install via Pelican egg + bundled JRE, accept
    EULA, start the server, verify it binds the TCP port (proves the JVM
    came up and the server is listening on the Minecraft protocol), stop
    and delete. EULA is accepted explicitly here for CI — the product
    itself never auto-accepts."""
    games = cli_run(device, 'games', 'list')
    candidates = [
        g for g in games
        if 'minecraft/java' in g.get('upstreamRef', '').lower()
        and g.get('tier') in ('verified', 'compatible')
    ]
    print('minecraft candidates ({}): {}'.format(
        len(candidates), [g['id'] for g in candidates[:10]]))
    assert candidates, 'no Minecraft Java entries in catalog with tier verified|compatible'
    candidates.sort(key=lambda g: (0 if g['id'] == 'paper' else 1, g['id']))
    g = candidates[0]
    print('using minecraft entry:', g['id'], 'name=', g['name'], 'ref=', g.get('upstreamRef'))

    port = g.get('defaultPort') or 25565
    cli_run(device, 'server', 'create', 'mc-real', g['id'], '--port', str(port))
    cli_run(device, 'server', 'install', 'mc-real')
    wait_status(device, 'mc-real', 'stopped', timeout=900)

    install_dir = '/data/games/servers/mc-real'
    out = device.run_ssh('ls {0} 2>&1'.format(install_dir))
    assert '.jar' in out, 'minecraft .jar missing post-install: ' + out

    device.run_ssh(
        'echo eula=true > {0}/eula.txt && chown games:games {0}/eula.txt'.format(install_dir))

    s = cli_run(device, 'server', 'start', 'mc-real')
    assert s['status'] == 'running'

    deadline = time.time() + 240
    bound = ''
    while time.time() < deadline:
        bound = device.run_ssh('ss -tlnp 2>/dev/null | grep -E "java|:{0}\\b" || true'.format(port))
        if 'java' in bound or ':{0}'.format(port) in bound:
            break
        time.sleep(5)
    assert 'java' in bound or ':{0}'.format(port) in bound, \
        'minecraft java server should be listening on tcp — ss output: {0!r}'.format(bound)

    lines = cli_run(device, 'server', 'logs', 'mc-real')
    print('mc server first 20 log lines:', lines[:20])

    cli_run(device, 'server', 'stop', 'mc-real')
    cli_run(device, 'server', 'delete', 'mc-real')

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

    again = device.run_ssh('games.cli server start stub 2>&1; echo EXIT=$?', throw=False)
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
