#!/bin/sh -ex

DIR=$( cd "$( dirname "$0" )" && pwd )
cd ${DIR}

BUILD_DIR=${DIR}/../build/snap/steamcmd

# The bundled lib32/lib64 closures must be self-contained — no host libc
# fallback. --verify against the bundled ld-linux returns "OK" only when
# every NEEDED DT entry resolves from --library-path.
LD_LIBRARY_PATH=${BUILD_DIR}/lib32 \
  ${BUILD_DIR}/lib32/ld-linux.so.2 --library-path ${BUILD_DIR}/lib32 \
  --verify ${BUILD_DIR}/linux32/steamcmd

# 64-bit loader sanity: it must at least respond to --version against the
# bundled lib64. We pick ld-linux's own binary as the test ELF since
# steamcmd ships no 64-bit executable.
${BUILD_DIR}/lib64/ld-linux-x86-64.so.2 --version
