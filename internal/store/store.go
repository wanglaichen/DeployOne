package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"webdown/internal/model"
)

type Store struct {
	mu       sync.RWMutex
	filePath string
	items    []model.APK
}

func New(filePath string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	s := &Store{filePath: filePath}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) List() []model.APK {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]model.APK, 0, len(s.items))
	for i := len(s.items) - 1; i >= 0; i-- {
		items = append(items, s.items[i])
	}
	return items
}

func (s *Store) ListByCategory(category string) []model.APK {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]model.APK, 0)
	for i := len(s.items) - 1; i >= 0; i-- {
		item := s.items[i]
		if item.ResolvedCategory() == category {
			items = append(items, item)
		}
	}
	return items
}

func (s *Store) CountByCategory() map[string]int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	counts := map[string]int{
		model.CategoryAndroid: 0,
		model.CategoryIOS:     0,
		model.CategoryFile:    0,
	}
	for _, item := range s.items {
		counts[item.ResolvedCategory()]++
	}
	return counts
}

func (s *Store) Get(id string) (model.APK, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, item := range s.items {
		if item.ID == id {
			return item, true
		}
	}
	return model.APK{}, false
}

func (s *Store) Add(apk model.APK) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.items = append(s.items, apk)
	return s.saveLocked()
}

func (s *Store) load() error {
	content, err := os.ReadFile(s.filePath)
	if errors.Is(err, os.ErrNotExist) {
		s.items = []model.APK{}
		return nil
	}
	if err != nil {
		return fmt.Errorf("read data file: %w", err)
	}
	if len(content) == 0 {
		s.items = []model.APK{}
		return nil
	}
	if err := json.Unmarshal(content, &s.items); err != nil {
		return fmt.Errorf("parse data file: %w", err)
	}
	return nil
}

func (s *Store) saveLocked() error {
	content, err := json.MarshalIndent(s.items, "", "  ")
	if err != nil {
		return fmt.Errorf("encode data file: %w", err)
	}
	if err := os.WriteFile(s.filePath, content, 0644); err != nil {
		return fmt.Errorf("write data file: %w", err)
	}
	return nil
}
