package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type ImageEntry struct {
	Data     []byte
	MIMEType string
}

type Store struct {
	mu      sync.RWMutex
	dataDir string
	images  map[string][]string // screenID -> on-disk file paths
	qrs     map[string][]byte
}

func NewStore(screenIDs []string, dataDir string) (*Store, error) {
	s := &Store{
		dataDir: dataDir,
		images:  make(map[string][]string, len(screenIDs)),
		qrs:     make(map[string][]byte, len(screenIDs)),
	}
	for _, id := range screenIDs {
		dir := filepath.Join(dataDir, id)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("creating dir for screen %s: %w", id, err)
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil, fmt.Errorf("reading dir for screen %s: %w", id, err)
		}

		s.images[id] = make([]string, 0, len(entries))
		for _, e := range entries {
			if !e.IsDir() {
				s.images[id] = append(s.images[id], filepath.Join(dir, e.Name()))
			}
		}
	}
	return s, nil
}

func (s *Store) SetQR(screenID string, data []byte) {
	s.mu.Lock()
	s.qrs[screenID] = data
	s.mu.Unlock()
}

func (s *Store) Add(screenID string, data []byte) error {
	dir := filepath.Join(s.dataDir, screenID)
	path := filepath.Join(dir, fmt.Sprintf("%d.bin", time.Now().UnixNano()))
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing image: %w", err)
	}
	s.mu.Lock()
	s.images[screenID] = append(s.images[screenID], path)
	s.mu.Unlock()
	return nil
}

// Random returns a random image for the screen, or the QR code if no images exist.
func (s *Store) Random(screenID string) (ImageEntry, bool) {
	s.mu.RLock()
	paths := s.images[screenID]
	var path string
	if len(paths) > 0 {
		path = paths[rand.Intn(len(paths))]
	}
	qr := s.qrs[screenID]
	s.mu.RUnlock()

	if path == "" {
		if qr == nil {
			return ImageEntry{}, false
		}
		return ImageEntry{Data: qr, MIMEType: "image/png"}, true
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ImageEntry{}, false
	}
	return ImageEntry{Data: data, MIMEType: http.DetectContentType(data)}, true
}

func (s *Store) HasScreen(screenID string) bool {
	s.mu.RLock()
	_, ok := s.images[screenID]
	s.mu.RUnlock()
	return ok
}

func (s *Store) Screens() []string {
	s.mu.RLock()
	out := make([]string, 0, len(s.images))
	for id := range s.images {
		out = append(out, id)
	}
	s.mu.RUnlock()
	return out
}
