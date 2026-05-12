#!/bin/bash
set -e
trap 'echo "[steamcmd.sh] failed at line $LINENO with exit $?" >&2' ERR

# /snap/ is squashfs (read-only); steamcmd self-updates `package/` and writes
# state next to its binary, so we copy the bundle to a writable runtime dir
# under $SNAP_DATA on first run. The lib32/ stays in /snap (doesn't change).
SCDIR=/snap/game-server/current/steamcmd
LIBS="${SCDIR}/lib32"
LD="${LIBS}/ld-linux.so.2"

RUNTIME=/var/snap/game-server/current/.steam-runtime
if [ ! -x "${RUNTIME}/linux32/steamcmd" ]; then
    mkdir -p "${RUNTIME}"
    # Copy everything from the bundled steamcmd dir except lib32/lib64 (which
    # stay in /snap — they don't mutate). Picks up linux32/, public/,
    # steamcmd.sh and any future tarball additions automatically. The
    # steamcmd binary was patchelf'd at build time to use
    # /snap/.../steamcmd/lib32/ld-linux.so.2 as its ELF interpreter, so it
    # works post-copy without any runtime fixup.
    for f in "${SCDIR}"/*; do
        name=$(basename "$f")
        case "$name" in
            lib32|lib64) continue ;;
        esac
        cp -r "$f" "${RUNTIME}/"
    done
    # steamcmd's ELF interpreter was patched at build time to point at
    # ${RUNTIME}/linux32/ld-linux.so.2 — create that as a symlink to the
    # actual ld-linux that lives in our /snap-bundled lib32.
    ln -sf "${LIBS}/ld-linux.so.2" "${RUNTIME}/linux32/ld-linux.so.2"
    # If we happen to be running as root (install hook, manual SSH diag, etc.)
    # ensure the runtime + game-server's HOME end up owned by game-server,
    # otherwise the backend service (which runs as game-server) can't write
    # back into them and steamcmd dies with permission errors / exit 1.
    if [ "$(id -u)" = "0" ]; then
        chown -RH game-server:game-server "${RUNTIME}"
        chown -R game-server:game-server "${HOME_OVERRIDE:-/var/snap/game-server/current/.steam-home}" 2>/dev/null || true
    fi
fi

export HOME="${HOME_OVERRIDE:-/var/snap/game-server/current/.steam-home}"
mkdir -p "${HOME}"

cd "${RUNTIME}"

# Library order matters: steamcmd ships its OWN linux32/libstdc++.so.6
# (2013-era 32-bit libstdc++ it was built against). The official
# steamcmd.sh prepends linux32/ to LD_LIBRARY_PATH so Steam's libstdc++
# wins over system libs. Mirror that — fall back to our bookworm lib32
# only for libs Steam doesn't ship (libc, libssl, libcurl, nss_*, ...).
export LD_LIBRARY_PATH="${RUNTIME}/linux32:${LIBS}:${LD_LIBRARY_PATH:-}"
export SSL_CERT_FILE="${SSL_CERT_FILE:-/etc/ssl/certs/ca-certificates.crt}"

# Sanity-check the patched interpreter actually exists. If it doesn't, the
# kernel will refuse the exec with a useless 'no such file' that the wrapper
# would otherwise swallow.
echo "[steamcmd.sh] binary: ${RUNTIME}/linux32/steamcmd" >&2
echo "[steamcmd.sh] interpreter: $(LD_LIBRARY_PATH= /usr/bin/file ${RUNTIME}/linux32/steamcmd 2>&1 || true)" >&2
echo "[steamcmd.sh] expected interpreter file at ${LIBS}/ld-linux.so.2:" >&2
ls -la "${LIBS}/ld-linux.so.2" >&2 || true

# Exec the binary DIRECTLY (no explicit ld-linux invocation). The binary's
# ELF interpreter was patchelf'd at build time to point at
# /snap/.../steamcmd/lib32/ld-linux.so.2 — that path exists once the snap
# is installed. This way /proc/self/exe correctly reports the steamcmd
# binary path (under writable RUNTIME), so when steamcmd derives STEAMROOT
# from /proc/self/exe and chdirs there, cwd is writable — fixes the EROFS
# that masquerades as 'Steam needs to be online'.
exec "${RUNTIME}/linux32/steamcmd" "$@"
