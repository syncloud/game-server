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
    # Copy everything from the bundled steamcmd dir except lib32/lib64
    # (those stay in /snap — they don't mutate). Picks up linux32/,
    # steamcmd.sh, steam.sh, and any future tarball additions.
    for f in "${SCDIR}"/*; do
        name=$(basename "$f")
        case "$name" in
            lib32|lib64) continue ;;
        esac
        cp -r "$f" "${RUNTIME}/"
    done
    if [ "$(id -u)" = "0" ]; then
        chown -R game-server:game-server "${RUNTIME}"
        chown -R game-server:game-server "${HOME_OVERRIDE:-/var/snap/game-server/current/.steam-home}" 2>/dev/null || true
    fi
fi

# Always (re)stage the ELF interpreter file next to the binary. steamcmd's
# PT_INTERP was patched at build time to /var/snap/.../runtime/linux32/
# ld-linux.so.2 — copy the actual loader there as a regular file (symlinks
# through /snap/<app>/current/ aren't traversed by the kernel during
# PT_INTERP load). Idempotent — runs every invocation so partial prior
# runs can't leave the binary unloadable.
cp -f "${LIBS}/ld-linux.so.2" "${RUNTIME}/linux32/ld-linux.so.2"

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
INTERP_TARGET="${RUNTIME}/linux32/ld-linux.so.2"
echo "[steamcmd.sh] binary: ${RUNTIME}/linux32/steamcmd $(stat -c '%U:%G %a' ${RUNTIME}/linux32/steamcmd 2>/dev/null || echo MISSING)" >&2
echo "[steamcmd.sh] interpreter target: ${INTERP_TARGET} $(stat -c '%U:%G %a %s bytes' ${INTERP_TARGET} 2>/dev/null || echo MISSING)" >&2
echo "[steamcmd.sh] runtime/linux32 listing:" >&2
ls -la "${RUNTIME}/linux32/" >&2 || true
echo "[steamcmd.sh] mount namespace inode: $(readlink /proc/self/ns/mnt)" >&2
# Read PT_INTERP straight from the ELF, no patchelf required.
# .interp section is a NUL-terminated string near the start of the file.
echo "[steamcmd.sh] PT_INTERP per dd+strings:" >&2
dd if="${RUNTIME}/linux32/steamcmd" bs=1 count=1024 2>/dev/null | strings -a -n 8 | head -3 >&2 || true
# Confirm kernel can stat the interpreter path EXACTLY as the binary has it
echo "[steamcmd.sh] readlink on interp target dir:" >&2
ls -lH /var/snap/game-server/current/.steam-runtime/linux32/ld-linux.so.2 >&2 || true
echo "[steamcmd.sh] resolved path:" >&2
realpath /var/snap/game-server/current/.steam-runtime/linux32/ld-linux.so.2 >&2 || true
echo "[steamcmd.sh] first 4 bytes of interpreter:" >&2
od -c -N 4 "${INTERP_TARGET}" >&2 || true
echo "[steamcmd.sh] running interpreter --version:" >&2
"${INTERP_TARGET}" --version >&2 || echo "[steamcmd.sh] interp exit: $?" >&2
echo "[steamcmd.sh] running /snap-bundled interpreter --version (control):" >&2
"${LIBS}/ld-linux.so.2" --version >&2 || echo "[steamcmd.sh] /snap interp exit: $?" >&2

# Exec the binary DIRECTLY (no explicit ld-linux invocation). The binary's
# ELF interpreter was patchelf'd at build time to point at
# /snap/.../steamcmd/lib32/ld-linux.so.2 — that path exists once the snap
# is installed. This way /proc/self/exe correctly reports the steamcmd
# binary path (under writable RUNTIME), so when steamcmd derives STEAMROOT
# from /proc/self/exe and chdirs there, cwd is writable — fixes the EROFS
# that masquerades as 'Steam needs to be online'.
exec "${RUNTIME}/linux32/steamcmd" "$@"
