package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/tracyhatemice/who/ddns"
)

func main() {
	// Parse flags
	var (
		port       string
		verbose    bool
		configPath string
	)
	flag.StringVar(&port, "port", "80", "Port number to listen on")
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose logging")
	flag.StringVar(&configPath, "config", "", "Path to config file (optional)")
	flag.Parse()

	// Load configuration
	cfg, err := LoadConfig(configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize DDNS dispatcher
	var ddnsDispatcher *ddns.Dispatcher
	if len(cfg.DDNS) > 0 {
		ddnsConfigs := make([]ddns.Config, len(cfg.DDNS))
		for i, entry := range cfg.DDNS {
			ddnsConfigs[i] = ddns.Config{
				Provider:  entry.Provider,
				Domain:    entry.Domain,
				IPVersion: entry.IPVersion,
				IAM:       entry.IAM,
				AccessKey: entry.AccessKey,
				SecretKey: entry.SecretKey,
				ZoneID:    entry.ZoneID,
				TTL:       entry.TTL,
			}
		}
		ddnsDispatcher = ddns.NewDispatcher(ddnsConfigs)
		log.Printf("DDNS: loaded %d entries", len(cfg.DDNS))
	}

	// Build who names set and pre-load IPs into store
	store := NewStore()
	whoNames := make(map[string]bool)
	for _, entry := range cfg.Who {
		if entry.IAM != "" {
			whoNames[entry.IAM] = true
			if entry.IP != "" {
				store.Set(entry.IAM, entry.IP)
			}
		}
	}
	if len(whoNames) > 0 {
		log.Printf("WHO: pre-loaded %d entries", len(whoNames))
	}

	// Create server with dependencies
	server := &Server{
		store:      store,
		ddns:       ddnsDispatcher,
		verbose:    verbose,
		configPath: configPath,
		whoNames:   whoNames,
		config:     cfg,
	}

	// Setup routes
	mux := http.NewServeMux()
	mux.HandleFunc("GET /whoami", server.withLogging(server.whoamiHandler))
	mux.HandleFunc("GET /iam/{name}", server.withLogging(server.iamHandler))
	mux.HandleFunc("GET /iam/{name}/{ip}", server.withLogging(server.iamHandler))
	mux.HandleFunc("GET /whois/{name}", server.withLogging(server.whoisHandler))

	log.Printf("Starting up on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
