package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"

	qrcode "github.com/skip2/go-qrcode"
)

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	baseURL := flag.String("base-url", "", "base URL for QR codes (e.g. http://192.168.1.10:8080)")
	screensFlag := flag.String("screens", "screen1,screen2", "comma-separated screen IDs")
	dataDir := flag.String("data-dir", "data", "directory to persist uploaded images")
	flag.Parse()

	if *baseURL == "" {
		*baseURL = "http://localhost" + *addr
	}

	var screenIDs []string
	for _, s := range strings.Split(*screensFlag, ",") {
		s = strings.TrimSpace(s)
		if s != "" {
			screenIDs = append(screenIDs, s)
		}
	}
	if len(screenIDs) == 0 {
		log.Fatal("no screen IDs configured")
	}

	store, err := NewStore(screenIDs, *dataDir)
	if err != nil {
		log.Fatalf("initializing store: %v", err)
	}

	for _, id := range screenIDs {
		uploadURL := fmt.Sprintf("%s/upload/%s", *baseURL, id)
		png, err := qrcode.Encode(uploadURL, qrcode.Medium, 512)
		if err != nil {
			log.Fatalf("generating QR for %s: %v", id, err)
		}
		store.SetQR(id, png)
		log.Printf("screen %q -> upload URL: %s", id, uploadURL)
	}

	h := &Handlers{store: store}

	mux := http.NewServeMux()
	mux.HandleFunc("/", h.Index)
	mux.HandleFunc("/screen/", h.Screen)
	mux.HandleFunc("/upload/", h.Upload)
	mux.HandleFunc("/api/image/", h.APIImage)
	mux.HandleFunc("/api/upload/", h.APIUpload)
	mux.HandleFunc("/api/qr/", h.APIQr)
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	log.Printf("listening on %s", *addr)
	log.Fatal(http.ListenAndServe(*addr, mux))
}
