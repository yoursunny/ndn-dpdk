#!/bin/bash
set -eo pipefail
source mk/cflags.sh
TESTCOUNT=${TESTCOUNT:-1}

GO=$(which go)
SUDO='sudo -E'
if [[ $(id -u) -eq 0 ]]; then
  SUDO=
fi

getTestPkg() {
  # determine $TESTPKG from $PKG
  TESTDIR=$1/$(basename $1)test
  if [[ -d $TESTDIR ]]; then
    echo $TESTDIR
  else
    echo $1
  fi
}

if [[ $# -eq 0 ]]; then
  # run all tests, optional filter in $MK_GOTEST_FILTER

  find -name '*_test.go' -printf '%h\n' | sort -u | sed -E "${MK_GOTEST_FILTER:-}" \
    | xargs -I{} $SUDO $GO test {} -count=$TESTCOUNT

elif [[ $# -eq 1 ]]; then
  # run tests in one package
  PKG=${1%/}
  TESTPKG=$(getTestPkg $PKG)

  $SUDO $GO test -cover -covermode count -coverpkg ./$PKG -coverprofile /tmp/gotest.cover ./$TESTPKG -v -count=$TESTCOUNT
  $SUDO chown $(id -u) /tmp/gotest.cover
  $GO tool cover -html /tmp/gotest.cover -o /tmp/gotest.cover.html

elif [[ $# -eq 2 ]]; then
  # run one test
  PKG=${1%/}
  TESTPKG=$(getTestPkg $PKG)
  TEST=$2

  if [[ ${TEST,,} != test* ]] && [[ ${TEST,,} != bench* ]] && [[ ${TEST,,} != example* ]]; then
    TEST='Test'${TEST}
  fi
  RUN='-run '$TEST
  if [[ ${TEST,,} == bench* ]]; then
    RUN=$RUN' -bench '$TEST
  fi
  $SUDO GODEBUG=cgocheck=2 $DBG $GO test ./$TESTPKG -v -count=$TESTCOUNT $RUN

elif [[ $# -eq 3 ]]; then
  # run one test with debug tool
  DBGTOOL=$1
  PKG=${2%/}
  TESTPKG=$(getTestPkg $PKG)
  TEST=$3

  if [[ $DBGTOOL == 'gdb' ]]; then DBG='gdb --silent --args'
  elif [[ $DBGTOOL == 'valgrind' ]]; then DBG='valgrind'
  else
    echo 'Unknown debug tool:' $1 >/dev/stderr
    exit 1
  fi

  $GO test -c ./$TESTPKG -o /tmp/gotest.exe
  $SUDO $DBG /tmp/gotest.exe -test.v -test.run 'Test'$TEST'.*'
else
  echo 'USAGE: mk/gotest.sh [debug-tool] [directory] [test-name]' >/dev/stderr
  exit 1
fi
