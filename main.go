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
	"sync"
	"time"
)

type GeoInfo struct {
	Country string `json:"country"`
	City    string `json:"city"`
	ISP     string `json:"isp"`
}

type TelemetryEvent struct {
	Timestamp string `json:"timestamp"`
	Protocol  string `json:"protocol"`
	SourceIP  string `json:"source_ip"`
	Port      int    `json:"target_port"`
	Payload   string `json:"payload"`
	Country   string `json:"country"`
	City      string `json:"city"`
	ISP       string `json:"isp"`
}

var (
	logFile  *os.File
	geoCache = make(map[string]GeoInfo)
	cacheMu  sync.Mutex
)

func getGeo(ip string) GeoInfo {
	// Check cache first to prevent API rate-limiting
	cacheMu.Lock()
	if info, exists := geoCache[ip]; exists {
		cacheMu.Unlock()
		return info
	}
	cacheMu.Unlock()

	// Fetch from free IP API
	url := fmt.Sprintf("http://ip-api.com/json/%s?fields=status,country,city,isp", ip)
	resp, err := http.Get(url)
	if err == nil {
		defer resp.Body.Close()
		var result struct {
			Status  string `json:"status"`
			Country string `json:"country"`
			City    string `json:"city"`
			ISP     string `json:"isp"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err == nil && result.Status == "success" {
			info := GeoInfo{Country: result.Country, City: result.City, ISP: result.ISP}
			// Save to cache
			cacheMu.Lock()
			geoCache[ip] = info
			cacheMu.Unlock()
			return info
		}
	}
	return GeoInfo{Country: "Unknown", City: "Unknown", ISP: "Unknown"}
}

func logEvent(protocol string, remoteAddr string, port int, payload string) {
	ip := strings.Split(remoteAddr, ":")[0]
	
	// Ignore local Docker network tests
	if ip == "127.0.0.1" || strings.HasPrefix(ip, "172.") {
		return
	}

	geo := getGeo(ip)

	event := TelemetryEvent{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Protocol:  protocol,
		SourceIP:  ip,
		Port:      port,
		Payload:   strings.TrimSpace(payload),
		Country:   geo.Country,
		City:      geo.City,
		ISP:       geo.ISP,
	}

	data, err := json.Marshal(event)
	if err != nil {
		return
	}

	fmt.Println(string(data))
	if logFile != nil {
		logFile.Write(append(data, '\n'))
	}
}

// 1. Fake Telnet
func handleTelnet(conn net.Conn) {
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(15 * time.Second))
	conn.Write([]byte("Ubuntu 24.04 LTS\r\nlogin: "))
	reader := bufio.NewReader(io.LimitReader(conn, 1024))
	username, _ := reader.ReadString('\n')
	conn.Write([]byte("Password: "))
	password, _ := reader.ReadString('\n')
	go logEvent("telnet", conn.RemoteAddr().String(), 23, fmt.Sprintf("user: %s | pass: %s", username, password))
	conn.Write([]byte("\r\nLogin incorrect\r\n"))
}

// 2. Fake Redis
func handleRedis(conn net.Conn) {
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(10 * time.Second))
	buf := make([]byte, 2048)
	n, err := conn.Read(buf)
	if err != nil {
		return
	}
	rawCmd := string(buf[:n])
	go logEvent("redis", conn.RemoteAddr().String(), 6379, rawCmd)
	if strings.Contains(strings.ToUpper(rawCmd), "PING") {
		conn.Write([]byte("+PONG\r\n"))
	} else if strings.Contains(strings.ToUpper(rawCmd), "INFO") {
		conn.Write([]byte("$72\r\nredis_version:7.0.0\r\nos:Linux\r\ntcp_port:6379\r\nconnected_clients:1\r\nrole:master\r\n\r\n"))
	} else {
		conn.Write([]byte("-ERR unknown command\r\n"))
	}
}

// 3. Fake Web Server
func startHTTPServer() {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		payload := fmt.Sprintf("%s %s (User-Agent: %s)", r.Method, r.URL.RequestURI(), r.UserAgent())
		go logEvent("http", r.RemoteAddr, 8080, payload)
		w.Header().Set("Server", "Apache/2.4.52 (Ubuntu)")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("<!DOCTYPE HTML PUBLIC \"-//IETF//DTD HTML 2.0//EN\">\n<html><head>\n<title>404 Not Found</title>\n</head><body>\n<h1>Not Found</h1>\n<p>The requested URL was not found on this server.</p>\n</body></html>"))
	})
	srv := &http.Server{Addr: ":8080", Handler: handler, ReadTimeout: 5 * time.Second, WriteTimeout: 5 * time.Second}
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
		if err == nil {
			go handler(conn)
		}
	}
}

func main() {
	var err error
	logFile, err = os.OpenFile("/logs/voidsink.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	defer logFile.Close()

	log.Println("[+] voidsink active. Listening on ports 23, 6379, 8080...")
	go startTCPListener(23, handleTelnet)
	go startTCPListener(6379, handleRedis)
	startHTTPServer()
}