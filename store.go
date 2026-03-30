package main

import (
	"math/rand"
	"sync"
)

type ImageEntry struct {
	Data     []byte
	MIMEType string
}

type Store struct {
	mu     sync.RWMutex
	images map[string][]ImageEntry
	qrs    map[string][]byte
}

func NewStore(screenIDs []string) *Store {
	s := &Store{
		images: make(map[string][]ImageEntry, len(screenIDs)),
		qrs:    make(map[string][]byte, len(screenIDs)),
	}
	for _, id := range screenIDs {
		s.images[id] = nil
	}
	return s
}

func (s *Store) SetQR(screenID string, data []byte) {
	s.mu.Lock()
	s.qrs[screenID] = data
	s.mu.Unlock()
}

func (s *Store) Add(screenID string, entry ImageEntry) {
	s.mu.Lock()
	s.images[screenID] = append(s.images[screenID], entry)
	s.mu.Unlock()
}

// Random returns a random image for the screen, or the QR code if no images exist.
func (s *Store) Random(screenID string) (ImageEntry, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	imgs := s.images[screenID]
	if len(imgs) == 0 {
		qr, ok := s.qrs[screenID]
		if !ok {
			return ImageEntry{}, false
		}
		return ImageEntry{Data: qr, MIMEType: "image/png"}, true
	}
	return imgs[rand.Intn(len(imgs))], true
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
