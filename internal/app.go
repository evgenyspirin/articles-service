package internal

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"articles-service/internal/articlesprocessor"
	"articles-service/internal/storage"
)

const maxLimit = 100

type App struct {
	logger     *zap.Logger
	proc       *articlesprocessor.ArticlesProcessor
	resultChan chan []string
}

func NewApp() (*App, error) {
	// logger
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("cannot initialize zap logger: %v", err)
	}
	defer logger.Sync()

	// pars run args
	var limit int
	flag.IntVar(&limit, "l", 0, "limit")
	flag.Parse()
	if limit == 0 {
		log.Fatal("please provide limit of articles")
	}
	if limit > maxLimit {
		log.Fatalf("max limit is out of range: %v", maxLimit)
	}

	// storage
	st := storage.New(logger, limit)
	// processor
	ap := articlesprocessor.New(logger, limit, st)

	return &App{
		logger:     logger,
		proc:       ap,
		resultChan: make(chan []string, 1),
	}, nil
}

func (a *App) Close() {
	if a.logger != nil {
		_ = a.logger.Sync()
	}
}

func (a *App) Run(ctx context.Context) error {
	a.logger.Info("running articles service...")

	// context with os signals cancel chan
	// any process/app/service must be able to shut down gracefully(avoid kill)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGUSR1)
	defer stop()

	// "errgroup" instead of "WaitGroup" because:
	// - allows return an error from goroutine
	// - group errors from multiple gorutines into one
	// - wg.Add(1), wg.Done() - automatically under the hood, so never catch deadlock if you forget something ;-)
	// - allows orchestration of parallel processes through the context.Context(gracefull shut down)
	//start := time.Now()
	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		articles, err := a.proc.TopArticles(ctx)
		if err != nil {
			return fmt.Errorf("ProcessArticles error: %w", err)
		}

		a.resultChan <- articles

		return nil
	})

	// waiting when processing finished or sigurg signal
	select {
	case <-ctx.Done():
	case topArticles := <-a.resultChan:
		for _, val := range topArticles {
			fmt.Println(val)
		}
	}

	if err := g.Wait(); err != nil {
		a.logger.Error("articles service returning an error", zap.Error(err))
		return err
	}

	a.logger.Info("articles service exited properly")

	return nil
}

func (a *App) Logger() *zap.Logger { return a.logger }
