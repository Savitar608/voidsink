# Voidsink: Lightweight Go Honeypot & Telemetry Pipeline

**Voidsink** is a low-interaction, high-performance honeypot written in Go, deployed alongside a complete observability stack (Loki, Promtail, Grafana). It safely simulates vulnerable network services, captures automated attacks (botnets, vulnerability scanners), enriches telemetry with GeoIP data, and visualizes the threat landscape in real-time.

---

## 🏗 Architecture

The system is deployed as a multi-container Docker Compose stack:

1. **Voidsink (Go Engine):** Compiled as a static binary into a distroless `scratch` container. Handles incoming TCP connections asynchronously using goroutines.
2. **Promtail:** Tails the raw JSON logs outputted by the Go engine, parses fields, and forwards them to Loki.
3. **Grafana Loki:** Acts as the central log aggregation database.
4. **Grafana:** Provides the real-time SOC dashboard using Dashboards as Code for zero-touch provisioning.

---

## ✨ Features

* **Simulated Vulnerable Services:**
  * **Port 23 (Telnet):** Simulates an Ubuntu login prompt to capture Mirai botnet brute-force credentials.
  * **Port 8080 (HTTP):** Simulates an Apache web server to capture web scanners (e.g., WordPress exploits, `.env` harvesting).
  * **Port 6379 (Redis):** Simulates an exposed Redis database to capture unauthorized access sweeps and crypto-mining attempts.
* **Smart GeoIP Enrichment:** Utilizes `ip-api.com` to resolve attacker IPs to their origin country, city, and ISP, protected by an in-memory mutex cache to prevent rate-limiting.
* **Asynchronous Logging:** File I/O and API lookups run in background goroutines to prevent blocking network listeners.
* **Persistent State:** Uses local volume mapping for Grafana's database (`grafana/data`) and provisioning files (`grafana/provisioning`).

---

## 🚀 Quick Start

### 1. Clone & Configure Permissions
Grafana runs as a non-root user (UID `472`). Grant ownership to the local directories before launching:

```bash
mkdir -p grafana/data grafana/provisioning/dashboards logs
sudo chown -R 472:472 grafana
sudo chmod -R 755 grafana
```

### 2. Launch the Stack

Use the orchestration script to build the Go binary and start containers:

```bash
./start.sh
```

### 3. Access the Dashboard

Navigate to `http://<your-server-ip>:3000` in your web browser.

* **Username:** `admin`
* **Default Password:** `admin` (You will be prompted to change this on first login).

---

## 🎯 Testing the Honeypot

Simulate an attack locally or via VPN:

**Test Web Scanners (Port 8080):**

```bash
curl -H "User-Agent: Nikto/2.1.6" http://<your-server-ip>:8080/.env
```

**Test Redis Sweep (Port 6379):**

```bash
echo -e "INFO\r\n" | nc -w 1 <your-server-ip> 6379
```

---

## 📁 Repository Structure

```text
voidsink/
├── docker-compose.yml
├── Dockerfile
├── main.go
├── start.sh
├── stop.sh
├── grafana/
│   ├── data/
│   └── provisioning/
├── logs/
└── promtail/
    └── promtail-config.yaml
```

---

## 🛑 Teardown

```bash
./stop.sh
```

---

## ⚠️ Disclaimer

Intended for educational, research, and defensive purposes only. Do not deploy on corporate networks without explicit authorization.