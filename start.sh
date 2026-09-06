#!/bin/bash
# start.sh - Launch the voidsink stack

# Ensure we are in the voidsink directory
cd "$(dirname "$0")" || exit 1

echo "[+] Starting voidsink deception engine and PLG telemetry stack..."
docker compose up -d --build

echo "[+] voidsink is running."
echo "[+] Grafana dashboard available at: http://<your-vm-public-ip>:3000"
