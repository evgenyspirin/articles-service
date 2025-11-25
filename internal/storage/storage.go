package storage

import (
	"container/heap"
	"sort"
	"sync"

	"go.uber.org/zap"
)

type (
	// Storage :first, we already have a predefined data structure size,
	// which is determined by the limit.
	// second, we don't need to retrieve sorted data during insertion (B-tree best in this case with bulk insertion),
	// we only need the sorted results at the end therefore "min-heap" is ideal.
	Storage struct {
		logger *zap.Logger
		limit  int
		// one reader at the end
		// therefore no sense of sync.RWMutex
		mu   sync.Mutex
		data MinHeap
	}
	Article struct {
		Name        string
		NumComments uint64
	}
)

func New(logger *zap.Logger, limit int) *Storage {
	h := NewMinHeap(limit)
	heap.Init(&h)

	return &Storage{
		logger: logger,
		limit:  limit,
		data:   h,
	}
}

func (s *Storage) Insert(a Article) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.data) < s.limit {
		heap.Push(&s.data, a)
		return
	}

	minElem := s.data[0]
	if a.NumComments <= minElem.NumComments {
		return
	}

	s.data[0] = a
	heap.Fix(&s.data, 0)
}

func (s *Storage) TopArticlesNames() []string {
	sort.Slice(s.data, func(i, j int) bool {
		return s.data[i].NumComments > s.data[j].NumComments
	})

	names := make([]string, len(s.data))
	for i, a := range s.data {
		names[i] = a.Name
	}

	// "Be kind, help GC"
	s.data = nil

	return names
}
