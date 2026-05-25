package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Link represents a navigation link
type Link struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	URL       string    `json:"url"`
	Icon      string    `json:"icon"`
	Category  string    `json:"category"`
	CreatedAt time.Time `json:"created_at"`
}

// LinksData is the storage structure
type LinksData struct {
	Links []Link `json:"links"`
}

// ScanResult represents a discovered service
type ScanResult struct {
	IP       string `json:"ip"`
	Port     int    `json:"port"`
	Protocol string `json:"protocol"`
	Service  string `json:"service"`
	Status   string `json:"status"` // "open", "closed", "timeout"
}

// ScanResponse is the API response for scan results
type ScanResponse struct {
	Results []ScanResult `json:"results"`
	Summary map[string]int `json:"summary"`
}

// Global scan state
var (
	scanResults []ScanResult
	scanMutex   sync.RWMutex
	isScanning  bool
	scanMutex2  sync.RWMutex
)

var dataDir string

func main() {
	// Determine data directory
	execPath, err := os.Executable()
	if err != nil {
		execPath = "."
	}
	dataDir = filepath.Join(filepath.Dir(execPath), "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Fatalf("Failed to create data dir: %v", err)
	}

	// Get port from args or default
	port := "8080"
	for i, arg := range os.Args {
		if arg == "-port" && i+1 < len(os.Args) {
			port = os.Args[i+1]
		}
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// CORS for local development
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// Serve static files
	r.Static("/assets", "./assets")
	r.StaticFile("/", "./index.html")

	// API routes
	api := r.Group("/api")
	{
		api.GET("/links", handleGetLinks)
		api.POST("/links", handleCreateLink)
		api.PUT("/links/:id", handleUpdateLink)
		api.DELETE("/links/:id", handleDeleteLink)
		api.POST("/scan", handleScan)
		api.GET("/scan/status", handleScanStatus)
	}

	addr := fmt.Sprintf(":%s", port)
	log.Printf("🚀 Homie starting on http://localhost:%s", port)
	log.Printf("📁 Data directory: %s", dataDir)
	r.Run(addr)
}

// getLinksFile returns the path to links.json
func getLinksFile() string {
	return filepath.Join(dataDir, "links.json")
}

// loadLinks reads links from JSON file
func loadLinks() ([]Link, error) {
	file := getLinksFile()
	data, err := os.ReadFile(file)
	if err != nil {
		if os.IsNotExist(err) {
			return []Link{}, nil
		}
		return nil, err
	}
	var ld LinksData
	if err := json.Unmarshal(data, &ld); err != nil {
		return nil, err
	}
	// Sort by created_at desc
	sort.Slice(ld.Links, func(i, j int) bool {
		return ld.Links[i].CreatedAt.After(ld.Links[j].CreatedAt)
	})
	return ld.Links, nil
}

// saveLinks writes links to JSON file
func saveLinks(links []Link) error {
	file := getLinksFile()
	ld := LinksData{Links: links}
	data, err := json.MarshalIndent(ld, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(file, data, 0644)
}

// guessService attempts to identify the service running on a port
func guessService(port int) (protocol, serviceName string) {
	serviceMap := map[int][2]string{
		80:     {"http", "Web"},
		443:    {"https", "Web (SSL)"},
		8080:   {"http", "Web (Alt)"},
		8443:   {"https", "Web (Alt SSL)"},
		3000:   {"http", "Node.js"},
		3001:   {"http", "Node.js"},
		5173:   {"http", "Vite Dev"},
		8000:   {"http", "Python/Django"},
		8888:   {"http", "Jupyter"},
		5000:   {"http", "Flask/Default"},
		5200:   {"http", "Portainer"},
		9000:   {"http", "SonarQube/Grafana"},
		9090:   {"http", "Prometheus"},
		9091:   {"http", "qBittorrent"},
		9092:   {"http", "Kadalu/S3"},
		32400:  {"http", "Plex"},
		32469:  {"http", "Plex (DLNA)"},
		8096:   {"http", "Jellyfin"},
		8920:   {"https", "Jellyfin (SSL)"},
		8989:   {"http", "Sonarr"},
		8686:   {"http", "Lidarr"},
		8787:   {"http", "Radarr"},
		7878:   {"http", "Radarr (Alt)"},
		8210:   {"http", "Bazarr"},
		8050:   {"http", "Jackett"},
		9117:   {"http", "Jackett (Alt)"},
		9696:   {"http", "Prowlarr"},
		6767:   {"http", "Prowlarr (Alt)"},
		3306:   {"mysql", "MySQL"},
		5432:   {"postgres", "PostgreSQL"},
		6379:   {"redis", "Redis"},
		27017:  {"mongodb", "MongoDB"},
		9200:   {"http", "Elasticsearch"},
		5601:   {"http", "Kibana"},
		1883:   {"mqtt", "MQTT"},
		5672:   {"amqp", "RabbitMQ"},
		15672:  {"http", "RabbitMQ (Admin)"},
		8081:   {"http", "Macrometa"},
		18789:  {"http", "OpenClaw"},
		10000:  {"http", "Webmin"},
		9094:   {"http", "Sorted│Sharry"},
		7777:   {"http", "FileBrowser"},
		8089:   {"http", "Home Assistant"},
		8123:   {"http", "Home Assistant (Alt)"},
		1880:   {"http", "Node-RED"},
		3001:   {"http", "ioBroker"},
		8083:   {"http", "HomeBridge"},
		8581:   {"http", "Mattermost"},
		5222:   {"xmpp", "XMPP"},
		5349:   {"xmpp", "XMPP (TLS)"},
		22:     {"ssh", "SSH"},
		2222:   {"ssh", "SSH (Alt)"},
		21:     {"ftp", "FTP"},
		2049:   {"nfs", "NFS"},
		445:    {"smb", "SMB"},
		139:    {"smb", "SMB (NetBIOS)"},
		68:     {"dhcp", "DHCP"},
		67:     {"dhcp", "DHCP (Server)"},
		53:     {"dns", "DNS"},
		123:    {"ntp", "NTP"},
		514:    {"syslog", "Syslog"},
		873:    {"rsync", "Rsync"},
		8123:   {"http", "Pi-hole"},
		80:     {"http", "CUPS"},
		631:    {"http", "CUPS"},
		3389:   {"rdp", "RDP"},
		5900:   {"vnc", "VNC"},
		6080:   {"http", "noVNC"},
		6000:   {"x11", "X11"},
		7681:   {"http", "Plexamp"},
		8925:   {"http", "Tautulli"},
		4749:   {"http", "Tautulli (Alt)"},
		32000:  {"http", "Digital Library"},
		82:     {"http", "Immich"},
		23000:  {"http", "Immich (Alt)"},
		8765:   {"http", "Bookstack"},
		2443:   {"http", "Bookstack (SSL)"},
		30000:  {"http", "Nextcloud"},
		9443:   {"https", "Nextcloud (SSL)"},
		9999:   {"http", "Urbackup"},
		41511:  {"http", "Urbackup (Alt)"},
		7878:   {"http", "Mylar"},
		8051:   {"http", " Mylar3"},
		8875:   {"http", "Calibre-Web"},
		8083:   {"http", "Calibre-Web (Alt)"},
		8181:   {"http", "Calibre-Web (SSL)"},
		1337:   {"http", "Wireguard"},
		51820:  {"udp", "Wireguard (UDP)"},
		51821:  {"udp", "Wireguard (Alt)"},
		8554:   {"http", "Jitsi"},
		5222:   {"http", "Jitsi (XMPP)"},
		5349:   {"http", "Jitsi (STUN)"},
		10080:  {"http", "Mumble"},
		64738:  {"udp", "Mumble (Voice)"},
		9093:   {"http", "Alertmanager"},
		3100:   {"http", "Grafana (Loki)"},
		4222:   {"http", "NATS"},
		8222:   {"http", "NATS (Alt)"},
		15691:  {"http", "NATS (Monitor)"},
		15692:  {"http", "NATS (Stats)"},
		8001:   {"http", "XZygote"},
		7700:   {"http", "uTorrent"},
		6881:  {"http", "Transmission"},
		9095:   {"http", "Transmission (Alt)"},
		30000:  {"http", "TrueNAS"},
		445:    {"smb", "TrueNAS (SMB)"},
		SSH:    {"ssh", ""},
	}

	if v, ok := serviceMap[port]; ok {
		return v[0], v[1]
	}

	// Default guesses based on port ranges
	switch {
	case port >= 80 && port <= 89:
		return "http", "Web"
	case port >= 8000 && port <= 8999:
		return "http", "Web Service"
	case port >= 5000 && port <= 5999:
		return "http", "API Service"
	case port >= 9000 && port <= 9999:
		return "http", "Management"
	default:
		return "http", fmt.Sprintf("Port %d", port)
	}
}

// scanPort checks if a single port is open on an IP
func scanPort(ip string, port int, timeout time.Duration) ScanResult {
	result := ScanResult{
		IP:   ip,
		Port: port,
	}

	protocol, service := guessService(port)
	result.Protocol = protocol
	result.Service = service

	// TCP scan
	addr := fmt.Sprintf("%s:%d", ip, port)
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		result.Status = "closed"
		return result
	}
	conn.Close()
	result.Status = "open"

	return result
}

// getLocalNetwork returns the local IP and network to scan
func getLocalNetwork() (string, string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", "", err
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && ipnet.IP.To4() != nil {
			ip := ipnet.IP.String()
			// Get the /24 network
			parts := strings.Split(ip, ".")
			if len(parts) == 4 {
				return ip, fmt.Sprintf("%s.%s.%s", parts[0], parts[1], parts[2]), nil
			}
		}
	}
	return "", "", fmt.Errorf("no suitable network found")
}

// commonPorts returns the list of ports to scan
func commonPorts() []int {
	return []int{
		22, 80, 443, 445, 3306, 5432, 6379, 8080, 8443, 9090,
		3000, 3001, 5000, 5173, 8000, 8888, 9000, 10000,
		1880, 1883, 32400, 32469, 8096, 8920, 8989, 8686,
		8787, 7878, 8210, 8050, 9117, 9696, 6767, 9091,
		9092, 9200, 5601, 5672, 15672, 8123, 1880, 8581,
		8081, 18789, 8222, 4222, 8222, 15691, 15692, 9093,
		3100, 8001, 8089, 3389, 5900, 68, 67, 53, 123,
		514, 873, 2049, 139, 631, 7777, 8925, 2443, 82,
		23000, 9443, 9999, 41511, 7878, 8051, 8875, 1337,
		51820, 8554, 5222, 5349, 10080, 64738, 5190,
	}
}

// handleScan performs network scan
func handleScan(c *gin.Context) {
	scanMutex2.Lock()
	if isScanning {
		scanMutex2.Unlock()
		c.JSON(http.StatusConflict, gin.H{"error": "Scan already in progress"})
		return
	}
	isScanning = true
	scanMutex2.Unlock()

	go runScan()

	c.JSON(http.StatusOK, gin.H{"message": "Scan started"})
}

func runScan() {
	defer func() {
		scanMutex2.Lock()
		isScanning = false
		scanMutex2.Unlock()
	}()

	localIP, network, err := getLocalNetwork()
	if err != nil {
		log.Printf("Scan failed: %v", err)
		return
	}
	log.Printf("🔍 Starting scan from %s, network: %s.x", localIP, network)

	ports := commonPorts()
	timeout := 500 * time.Millisecond
	var wg sync.WaitGroup
	var mutex sync.Mutex
	results := []ScanResult{}
	resultsMap := make(map[string]ScanResult) // key: ip:port, dedup

	// Scan all IPs in /24 range
	for i := 1; i <= 254; i++ {
		ip := fmt.Sprintf("%s.%d", network, i)
		wg.Add(1)
		go func(ip string) {
			defer wg.Done()
			for _, port := range ports {
				result := scanPort(ip, port, timeout)
				if result.Status == "open" {
					mutex.Lock()
					key := fmt.Sprintf("%s:%d", ip, port)
					resultsMap[key] = result
					mutex.Unlock()
				}
			}
		}(ip)
	}

	wg.Wait()

	// Convert map to slice
	for _, r := range resultsMap {
		results = append(results, r)
	}

	// Sort by IP then port
	sort.Slice(results, func(i, j int) bool {
		if results[i].IP == results[j].IP {
			return results[i].Port < results[j].Port
		}
		return results[i].IP < results[j].IP
	})

	// Build summary
	summary := make(map[string]int)
	for _, r := range results {
		summary[r.Service]++
	}

	scanMutex.Lock()
	scanResults = results
	scanMutex.Unlock()

	log.Printf("✅ Scan complete: %d open ports found", len(results))
}

// handleScanStatus returns current scan results
func handleScanStatus(c *gin.Context) {
	scanMutex.RLock()
	results := scanResults
	scanMutex.RUnlock()

	scanMutex2.RLock()
	scanning := isScanning
	scanMutex2.RUnlock()

	summary := make(map[string]int)
	for _, r := range results {
		summary[r.Service]++
	}

	c.JSON(http.StatusOK, gin.H{
		"scanning": scanning,
		"count":    len(results),
		"results":  results,
		"summary":  summary,
	})
}

// handleGetLinks returns all links
func handleGetLinks(c *gin.Context) {
	links, err := loadLinks()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, links)
}

// handleCreateLink creates a new link
func handleCreateLink(c *gin.Context) {
	var link Link
	if err := c.ShouldBindJSON(&link); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	link.ID = uuid.New().String()
	link.CreatedAt = time.Now()

	links, err := loadLinks()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	links = append(links, link)
	if err := saveLinks(links); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, link)
}

// handleUpdateLink updates a link
func handleUpdateLink(c *gin.Context) {
	id := c.Param("id")

	var link Link
	if err := c.ShouldBindJSON(&link); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	links, err := loadLinks()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	found := false
	for i, l := range links {
		if l.ID == id {
			links[i].Name = link.Name
			links[i].URL = link.URL
			links[i].Icon = link.Icon
			links[i].Category = link.Category
			found = true
			break
		}
	}

	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "Link not found"})
		return
	}

	if err := saveLinks(links); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, link)
}

// handleDeleteLink deletes a link
func handleDeleteLink(c *gin.Context) {
	id := c.Param("id")

	links, err := loadLinks()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	found := false
	for i, l := range links {
		if l.ID == id {
			links = append(links[:i], links[i+1:]...)
			found = true
			break
		}
	}

	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "Link not found"})
		return
	}

	if err := saveLinks(links); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Deleted"})
}
