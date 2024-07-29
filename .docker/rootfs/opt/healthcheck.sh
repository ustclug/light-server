#!/bin/sh

check_process() {
  local process_name=$1
  
  if pgrep "$process_name" > /dev/null; then
    return 0
  else
    exit 1
  fi
}

check_process "squid"
check_process "crond"
