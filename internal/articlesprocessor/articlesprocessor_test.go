package articlesprocessor

import (
	"context"
	"testing"
	"time"

	"articles-service/internal/articlesapi"
	"articles-service/internal/storage"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

// helpers
func strPtr(s string) *string { return &s }
func intPtr(i int) *int       { return &i }

func TestArticlesProcessor_TopArticles_Success(t *testing.T) {
	logger := zap.NewNop()

	const limit = 5

	st := storage.New(logger, limit)

	p := New(logger, limit, st)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	top, err := p.TopArticles(ctx)
	require.NoError(t, err)

	assert.NotEmpty(t, top, "TopArticles should return at least one article")

	assert.LessOrEqual(t, len(top), limit)
}

func TestArticlesProcessor_TopArticles_ContextCanceled(t *testing.T) {
	logger := zap.NewNop()

	st := storage.New(logger, 5)
	p := New(logger, 5, st)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	top, err := p.TopArticles(ctx)

	require.Error(t, err)
	assert.Nil(t, top)
}

func TestArticlesProcessor_processArticle(t *testing.T) {
	logger := zap.NewNop()

	type testCase struct {
		name         string
		input        *articlesapi.Article
		storageLimit int
		wantNames    []string
	}

	tests := []testCase{
		{
			name: "uses Title when present and NumComments set",
			input: &articlesapi.Article{
				Title:       strPtr("title-1"),
				StoryTitle:  strPtr("story-title-ignored"),
				NumComments: intPtr(10),
			},
			storageLimit: 10,
			wantNames:    []string{"title-1"},
		},
		{
			name: "uses StoryTitle when Title is nil and NumComments set",
			input: &articlesapi.Article{
				Title:       nil,
				StoryTitle:  strPtr("story-title-1"),
				NumComments: intPtr(5),
			},
			storageLimit: 10,
			wantNames:    []string{"story-title-1"},
		},
		{
			name: "skips when both Title and StoryTitle are nil",
			input: &articlesapi.Article{
				Title:       nil,
				StoryTitle:  nil,
				NumComments: intPtr(5),
			},
			storageLimit: 10,
			wantNames:    []string{},
		},
		{
			name: "skips when NumComments is nil even if Title present",
			input: &articlesapi.Article{
				Title:       strPtr("should-be-skipped"),
				StoryTitle:  nil,
				NumComments: nil,
			},
			storageLimit: 10,
			wantNames:    []string{},
		},
		{
			name: "prefers Title over StoryTitle if both present",
			input: &articlesapi.Article{
				Title:       strPtr("title-preferred"),
				StoryTitle:  strPtr("story-not-used"),
				NumComments: intPtr(3),
			},
			storageLimit: 10,
			wantNames:    []string{"title-preferred"},
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			st := storage.New(logger, tt.storageLimit)

			p := &ArticlesProcessor{
				logger:  logger,
				limit:   tt.storageLimit,
				storage: st,
			}

			err := p.processArticle(tt.input)
			require.NoError(t, err)

			names := st.TopArticlesNames()
			assert.Equal(t, tt.wantNames, names)
		})
	}
}

func TestArticlesProcessor_sendPagesToProcess(t *testing.T) {
	logger := zap.NewNop()

	type testCase struct {
		name          string
		pages         int
		expectedPages []int
	}

	tests := []testCase{
		{
			name:          "pages == 1 -> nothing sent, channel closed",
			pages:         1,
			expectedPages: []int{},
		},
		{
			name:          "pages == 2 -> sends only page 2",
			pages:         2,
			expectedPages: []int{2},
		},
		{
			name:          "pages == 5 -> sends 5..2 (skips first page)",
			pages:         5,
			expectedPages: []int{5, 4, 3, 2},
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			p := &ArticlesProcessor{
				logger: logger,
				in:     make(InChan, tt.pages+1),
			}

			var g errgroup.Group
			p.sendPagesToProcess(tt.pages, &g)

			var got []int
			for page := range p.in {
				got = append(got, page)
			}
			if got == nil {
				got = []int{}
			}

			err := g.Wait()
			require.NoError(t, err)

			assert.Equal(t, tt.expectedPages, got)
		})
	}
}

func TestArticlesProcessor_sendPagesToProcess_IntegrationWithProducerStop(t *testing.T) {
	logger := zap.NewNop()

	p := &ArticlesProcessor{
		logger: logger,
		in:     make(InChan, 3),
		out:    make(OutChan, 3),
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var g errgroup.Group
	g.Go(func() error {
		for {
			select {
			case <-ctx.Done():
				return nil
			case _, ok := <-p.in:
				if !ok {
					return nil
				}
			}
		}
	})

	pages := 3
	p.sendPagesToProcess(pages, &g)

	for range p.in {
	}

	err := g.Wait()
	require.NoError(t, err)
}
