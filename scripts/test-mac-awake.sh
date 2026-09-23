#!/bin/bash
# What each line of `pmset -g pslog` does to the lid. The daemon itself needs
# root and a power cable to exercise; this is the part that decides.
set -uo pipefail
cd "$(dirname "$0")"
MAC_AWAKE_LIB=1 . ./mac-awake

fail=0
check() { # <what> <expected> <line>
  got=$(lid_setting "$3")
  if [ "$got" = "$2" ]; then echo "ok   $1"; else echo "FAIL $1: wanted '$2', got '$got'"; fail=1; fi
}

check "plugged in keeps the lid from sleeping it" 1 "Now drawing from 'AC Power'"
check "on battery the lid sleeps it"              0 "Now drawing from 'Battery Power'"
check "a UPS counts as battery"                   0 "Now drawing from 'UPS Power'"
check "the battery detail line decides nothing"   "" " -InternalBattery-0 (id=7602275)	100%; charged; 0:00 remaining present: true"
check "a blank line decides nothing"              "" ""
check "an unfamiliar line decides nothing"        "" "Now drawing from 'Something New'"

[ $fail = 0 ] && echo "all good"
exit $fail
