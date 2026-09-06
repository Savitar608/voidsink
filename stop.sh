#!/bin/bash
# stop.sh - Teardown the voidsink stack

# Ensure we are in the voidsink directory
cd "$(dirname "$0")" || exit 1

echo "[-] Stopping voidsink stack..."
docker compose down

echo "[-] voidsink has been stopped and networks cleared."
