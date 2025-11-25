package storage

import (
	"container/heap"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMinHeap(t *testing.T) {
	tests := []struct {
		name     string
		capacity int
	}{
		{name: "zero capacity", capacity: 0},
		{name: "small capacity", capacity: 1},
		{name: "normal capacity", capacity: 10},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			h := NewMinHeap(tt.capacity)

			assert.Equal(t, 0, h.Len(), "new heap must be empty")
			assert.GreaterOrEqual(t, cap(h), tt.capacity)
		})
	}
}

func TestMinHeap_HeapInterfaceOrder(t *testing.T) {
	tests := []struct {
		name      string
		articles  []Article
		wantOrder []uint64
	}{
		{
			name: "already sorted ascending",
			articles: []Article{
				{Name: "a", NumComments: 1},
				{Name: "b", NumComments: 2},
				{Name: "c", NumComments: 3},
			},
			wantOrder: []uint64{1, 2, 3},
		},
		{
			name: "random order",
			articles: []Article{
				{Name: "a", NumComments: 10},
				{Name: "b", NumComments: 5},
				{Name: "c", NumComments: 7},
				{Name: "d", NumComments: 1},
			},
			wantOrder: []uint64{1, 5, 7, 10},
		},
		{
			name: "duplicates",
			articles: []Article{
				{Name: "a", NumComments: 5},
				{Name: "b", NumComments: 5},
				{Name: "c", NumComments: 3},
			},
			wantOrder: []uint64{3, 5, 5},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			h := NewMinHeap(len(tt.articles))

			for _, a := range tt.articles {
				heap.Push(&h, a)
			}

			require.Equal(t, len(tt.wantOrder), h.Len())

			var got []uint64
			for h.Len() > 0 {
				v := heap.Pop(&h).(Article)
				got = append(got, v.NumComments)
			}

			assert.Equal(t, tt.wantOrder, got)
		})
	}
}

func TestMinHeap_LenLessSwap(t *testing.T) {
	t.Run("Len reflects number of elements", func(t *testing.T) {
		h := NewMinHeap(3)
		assert.Equal(t, 0, h.Len())

		h.Push(Article{Name: "a", NumComments: 1})
		assert.Equal(t, 1, h.Len())

		h.Push(Article{Name: "b", NumComments: 2})
		assert.Equal(t, 2, h.Len())
	})

	t.Run("Less compares by NumComments", func(t *testing.T) {
		h := MinHeap{
			{Name: "a", NumComments: 10},
			{Name: "b", NumComments: 5},
		}

		assert.False(t, h.Less(0, 1))
		assert.True(t, h.Less(1, 0))
	})

	t.Run("Swap swaps elements", func(t *testing.T) {
		h := MinHeap{
			{Name: "a", NumComments: 1},
			{Name: "b", NumComments: 2},
		}

		h.Swap(0, 1)

		require.Equal(t, 2, h.Len())
		assert.Equal(t, uint64(2), h[0].NumComments)
		assert.Equal(t, uint64(1), h[1].NumComments)
	})
}

func TestMinHeap_PushPopRaw(t *testing.T) {
	t.Run("Push and Pop single element", func(t *testing.T) {
		h := NewMinHeap(1)

		h.Push(Article{Name: "a", NumComments: 42})
		require.Equal(t, 1, h.Len())

		x := h.Pop().(Article)
		assert.Equal(t, "a", x.Name)
		assert.Equal(t, uint64(42), x.NumComments)
		assert.Equal(t, 0, h.Len())
	})

	t.Run("Pop from multiple elements (LIFO on raw Pop)", func(t *testing.T) {
		h := NewMinHeap(3)
		h.Push(Article{Name: "a", NumComments: 1})
		h.Push(Article{Name: "b", NumComments: 2})
		h.Push(Article{Name: "c", NumComments: 3})

		x1 := h.Pop().(Article)
		x2 := h.Pop().(Article)
		x3 := h.Pop().(Article)

		assert.Equal(t, "c", x1.Name)
		assert.Equal(t, "b", x2.Name)
		assert.Equal(t, "a", x3.Name)
		assert.Equal(t, 0, h.Len())
	})
}
