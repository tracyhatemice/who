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

	http.HandleFunc("/", handle(whoamiHandler, verbose))
	http.HandleFunc("/iam/", handle(iamHandler, verbose))
	http.HandleFunc("/whois/", handle(whoisHandler, verbose))

	log.Printf("Starting up on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
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
	name := r.URL.Path[len("/iam/"):]
	if name == "" {
		http.Error(w, "name required", http.StatusBadRequest)
		return
	}

	realIP := r.Header.Get("X-Real-Ip")
	store[name] = realIP
	_, _ = fmt.Fprintln(w, realIP)
}

func whoisHandler(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Path[len("/whois/"):]
	if name == "" {
		http.Error(w, "name required", http.StatusBadRequest)
		return
	}

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
