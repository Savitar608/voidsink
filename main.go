package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

// TelemetryEvent represents a standardized security event
type TelemetryEvent struct {
	Timestamp string `json:"timestamp"`
	Protocol  string `json:"protocol"`
	SourceIP  string `json:"source_ip"`
	Port      int    `json:"target_port"`
	Payload   string `json:"payload"`
}

var logFile *os.File

func logEvent(protocol string, remoteAddr string, port int, payload string) {
	ip := strings.Split(remoteAddr, ":")[0]
	event := TelemetryEvent{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Protocol:  protocol,
		SourceIP:  ip,
		Port:      port,
		Payload:   strings.TrimSpace(payload),
	}

	data, err := json.Marshal(event)
	if err != nil {
		return
	}

	// Write to both stdout and telemetry file
	fmt.Println(string(data))
	if logFile != nil {
		logFile.Write(append(data, '\n'))
	}
}

// 1. Fake Telnet (Port 23) - Catches IoT/Mirai brute-force credential sprays
func handleTelnet(conn net.Conn) {
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(15 * time.Second))

	// Send generic Linux login prompt
	conn.Write([]byte("Ubuntu 24.04 LTS\r\nlogin: "))

	reader := bufio.NewReader(io.LimitReader(conn, 1024))
	username, _ := reader.ReadString('\n')

	conn.Write([]byte("Password: "))
	password, _ := reader.ReadString('\n')

	logEvent("telnet", conn.RemoteAddr().String(), 23, fmt.Sprintf("user: %s | pass: %s", username, password))
	conn.Write([]byte("\r\nLogin incorrect\r\n"))
}

// 2. Fake Redis (Port 6379) - Catches crypto-miner droppers and INFO sweeps
func handleRedis(conn net.Conn) {
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(10 * time.Second))

	buf := make([]byte, 2048)
	n, err := conn.Read(buf)
	if err != nil {
		return
	}

	rawCmd := string(buf[:n])
	logEvent("redis", conn.RemoteAddr().String(), 6379, rawCmd)

	// Respond with realistic Redis errors or version info
	if strings.Contains(strings.ToUpper(rawCmd), "PING") {
		conn.Write([]byte("+PONG\r\n"))
	} else if strings.Contains(strings.ToUpper(rawCmd), "INFO") {
		conn.Write([]byte("$72\r\nredis_version:7.0.0\r\nos:Linux\r\ntcp_port:6379\r\nconnected_clients:1\r\nrole:master\r\n\r\n"))
	} else {
		conn.Write([]byte("-ERR unknown command\r\n"))
	}
}

// 3. Fake Web/Admin Server (Port 8080) - Catches vulnerability scans (.env, path traversal, exploit payloads)
func startHTTPServer() {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		payload := fmt.Sprintf("%s %s (User-Agent: %s)", r.Method, r.URL.RequestURI(), r.UserAgent())
		logEvent("http", r.RemoteAddr, 8080, payload)

		// Deceptive 404/mock header
		w.Header().Set("Server", "Apache/2.4.52 (Ubuntu)")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("<!DOCTYPE HTML PUBLIC \"-//IETF//DTD HTML 2.0//EN\">\n<html><head>\n<title>404 Not Found</title>\n</head><body>\n<h1>Not Found</h1>\n<p>The requested URL was not found on this server.</p>\n</body></html>"))
	})

	srv := &http.Server{
		Addr:         ":8080",
		Handler:      handler,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}
	log.Fatal(srv.ListenAndServe())
}

func startTCPListener(port int, handler func(net.Conn)) {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("Failed to bind port %d: %v", port, err)
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		go handler(conn)
	}
}

func main() {
	var err error
	logFile, err = os.OpenFile("/logs/voidsink.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	defer logFile.Close()

	log.Println("[+] Honeypot active. Listening on ports 23 (Telnet), 6379 (Redis), 8080 (HTTP)...")

	go startTCPListener(23, handleTelnet)
	go startTCPListener(6379, handleRedis)
	startHTTPServer()
}