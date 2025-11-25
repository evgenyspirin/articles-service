package articlesprocessor

import (
	"context"
	"errors"
	"runtime"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"articles-service/internal/articlesapi"
	"articles-service/internal/storage"
)

type (
	ArticlesProcessor struct {
		logger      *zap.Logger
		limit       int
		articlesAPI *articlesapi.Client
		in          InChan
		out         OutChan
		storage     *storage.Storage
	}
	OutChan = chan articlesapi.Articles
	InChan  = chan int
)

func New(
	logger *zap.Logger,
	limit int,
	storage *storage.Storage,
) *ArticlesProcessor {
	return &ArticlesProcessor{
		logger:      logger,
		limit:       limit,
		articlesAPI: articlesapi.New(logger),
		// small buffer to avoid potential blocking
		// "Rely on metrics, not guesses."
		out:     make(OutChan, articlesapi.MaxRPSPerCurrentHost),
		in:      make(InChan, articlesapi.MaxRPSPerCurrentHost),
		storage: storage,
	}
}

func (p *ArticlesProcessor) TopArticles(ctx context.Context) ([]string, error) {
	g, ctx := errgroup.WithContext(ctx)
	if err := p.runPipeline(ctx, g); err != nil {
		return nil, err
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return p.storage.TopArticlesNames(), nil
}

func (p *ArticlesProcessor) runPipeline(ctx context.Context, g *errgroup.Group) error {
	firstPage, err := p.articlesAPI.FetchPage(ctx, 1)
	if err != nil {
		return err
	}
	if firstPage == nil {
		return errors.New("no api data found")
	}
	p.out <- firstPage.Data

	p.runArticlesConsumer(ctx, g)
	p.runArticlesFetcherPool(ctx, g)
	p.sendPagesToProcess(firstPage.TotalPages, g)

	return nil
}

func (p *ArticlesProcessor) runArticlesConsumer(ctx context.Context, g *errgroup.Group) {
	p.logger.Info("starting ArticlesConsumer worker gracefully stopped")

	g.Go(func() error {
		defer func() {
			p.logger.Info("ArticlesConsumer worker gracefully stopped")
		}()
		for {
			select {
			case <-ctx.Done():
				return nil
			case articles, ok := <-p.out:
				if !ok {
					return nil
				}

				for _, a := range articles {
					if err := p.processArticle(a); err != nil {
						return err
					}
				}
			}
		}
	})
}

func (p *ArticlesProcessor) runArticlesFetcherPool(ctx context.Context, g *errgroup.Group) {
	p.logger.Info("starting ArticlesFetcher pool")

	g.Go(func() error {
		subG, subCtx := errgroup.WithContext(ctx)
		// "Rely on metrics, not guesses."
		for i := 0; i < runtime.NumCPU()*2; i++ {
			subG.Go(func() error {
				if err := p.producer(subCtx); err != nil {
					return err
				}
				return nil
			})
		}

		err := subG.Wait()
		close(p.out)

		p.logger.Info("ArticlesFetcher pool gracefully stopped")

		return err
	})
}

func (p *ArticlesProcessor) producer(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case page, ok := <-p.in:
			if !ok {
				return nil
			}

			resp, err := p.articlesAPI.FetchPage(ctx, page)
			if err != nil {
				return err
			}
			if resp == nil {
				break
			}

			p.out <- resp.Data
		}
	}
}

func (p *ArticlesProcessor) sendPagesToProcess(pages int, g *errgroup.Group) {
	g.Go(func() error {
		defer close(p.in)

		for pages != 0 {
			// skipping first page
			if pages == 1 {
				break
			}
			p.in <- pages
			pages--
		}
		return nil
	})
}

func (p *ArticlesProcessor) processArticle(article *articlesapi.Article) error {
	a := storage.Article{}

	if article.Title != nil {
		a.Name = *article.Title
	} else if article.Title == nil && article.StoryTitle != nil {
		a.Name = *article.StoryTitle
	} else if article.Title == nil && article.StoryTitle == nil {
		return nil
	}

	if article.NumComments == nil {
		return nil
	}
	a.NumComments = uint64(*article.NumComments)

	p.storage.Insert(a)

	return nil
}
