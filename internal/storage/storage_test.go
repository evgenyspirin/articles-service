package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestStorage_InsertAndTopArticlesNames(t *testing.T) {
	type testCase struct {
		name          string
		limit         int
		articles      []Article
		expectedNames []string
	}

	tests := []testCase{
		{
			name:  "limit bigger than number of articles",
			limit: 5,
			articles: []Article{
				{Name: "a", NumComments: 5},
				{Name: "b", NumComments: 10},
				{Name: "c", NumComments: 7},
			},
			expectedNames: []string{"b", "c", "a"},
		},
		{
			name:  "limit smaller than number of articles, keeps only top N",
			limit: 3,
			articles: []Article{
				{Name: "a", NumComments: 5},
				{Name: "b", NumComments: 10},
				{Name: "c", NumComments: 7},
				{Name: "d", NumComments: 2},
				{Name: "e", NumComments: 8},
			},
			expectedNames: []string{"b", "e", "c"},
		},
	}

	logger := zap.NewNop()

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			s := New(logger, tt.limit)

			for _, a := range tt.articles {
				s.Insert(a)
			}

			got := s.TopArticlesNames()
			assert.Equal(t, tt.expectedNames, got)
		})
	}
}

func TestStorage_Insert_IgnoresSmallerOrEqualThanMin(t *testing.T) {
	logger := zap.NewNop()
	s := New(logger, 2)

	// заполняем до лимита
	s.Insert(Article{Name: "a", NumComments: 5})
	s.Insert(Article{Name: "b", NumComments: 10})

	s.Insert(Article{Name: "c", NumComments: 3})
	s.Insert(Article{Name: "d", NumComments: 5})

	names := s.TopArticlesNames()
	require.Len(t, names, 2)
	assert.Equal(t, []string{"b", "a"}, names)
}

func TestStorage_Insert_ReplacesMinWhenBigger(t *testing.T) {
	logger := zap.NewNop()
	s := New(logger, 2)

	s.Insert(Article{Name: "a", NumComments: 5})
	s.Insert(Article{Name: "b", NumComments: 10})

	s.Insert(Article{Name: "c", NumComments: 7})

	names := s.TopArticlesNames()
	require.Len(t, names, 2)
	assert.Equal(t, []string{"b", "c"}, names)
}

func TestStorage_TopArticlesNames_EmptyStorage(t *testing.T) {
	logger := zap.NewNop()
	s := New(logger, 10)

	names := s.TopArticlesNames()
	assert.Empty(t, names)
}

func TestStorage_TopArticlesNames_ClearsData(t *testing.T) {
	logger := zap.NewNop()
	s := New(logger, 3)

	s.Insert(Article{Name: "a", NumComments: 1})
	s.Insert(Article{Name: "b", NumComments: 2})

	names := s.TopArticlesNames()
	require.NotEmpty(t, names)

	assert.Nil(t, s.data)
	assert.Equal(t, 0, len(s.data))
}

func TestStorage_ZeroLimit_PanicsOnInsert(t *testing.T) {
	logger := zap.NewNop()
	s := New(logger, 0)

	require.Panics(t, func() {
		s.Insert(Article{Name: "a", NumComments: 5})
	})
}
