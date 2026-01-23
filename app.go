package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

var (
	port    string
	verbose bool
	store   = make(map[string]string)
)

func init() {
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose logging")
	flag.StringVar(&port, "port", getEnv("WHOAMI_PORT_NUMBER", "80"), "give me a port number")
}

func main() {
	flag.Parse()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /whoami", handle(whoamiHandler, verbose))
	mux.HandleFunc("GET /iam/{name}", handle(iamHandler, verbose))
	mux.HandleFunc("GET /whois/{name}", handle(whoisHandler, verbose))

	log.Printf("Starting up on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

func handle(next http.HandlerFunc, verbose bool) http.HandlerFunc {
	if !verbose {
		return next
	}

	return func(w http.ResponseWriter, r *http.Request) {
		next(w, r)
		log.Printf("%s - - [%s] \"%s %s %s\" - -", r.RemoteAddr, time.Now().Format("02/Jan/2006:15:04:05 -0700"), r.Method, r.URL.Path, r.Proto)
	}
}

func whoamiHandler(w http.ResponseWriter, r *http.Request) {
	if realIP := r.Header.Get("X-Real-Ip"); realIP != "" {
		_, _ = fmt.Fprintln(w, realIP)
	}
}

func iamHandler(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	realIP := r.Header.Get("X-Real-Ip")
	if realIP == "" {
		http.Error(w, "X-Real-Ip header required", http.StatusBadRequest)
		return
	}

	store[name] = realIP
	_, _ = fmt.Fprintln(w, realIP)
}

func whoisHandler(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	realIP, ok := store[name]
	if !ok {
		http.NotFound(w, r)
		return
	}
	_, _ = fmt.Fprintln(w, realIP)
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
