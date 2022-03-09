package pool

import (
	"context"
	"fmt"
	"golang.org/x/sync/errgroup"
	"log"
)

type DeleterPoolT struct {
	Input chan ToDeleteItem
	g     *errgroup.Group
	ctx   context.Context
	ErrCh chan error
}

type ToDeleteItem struct {
	UserID  string
	ShortID string
}

type Deleter interface {
	SetDeleted(ctx context.Context, item ToDeleteItem) error
}

func New(ctx context.Context, repo Deleter) DeleterPoolT {
	input := make(chan ToDeleteItem, 1000)
	g, ctx := errgroup.WithContext(ctx)
	errCh := make(chan error)
	pool := DeleterPoolT{
		Input: input,
		g:     g,
		ctx:   ctx,
		ErrCh: errCh,
	}
	go pool.Run(repo)
	return pool
}

func (p DeleterPoolT) Run(repo Deleter) {
	numWorkers := 4

	for i := 0; i < numWorkers; i++ {
		p.g.Go(func() error {
			for {
				select {
				case item := <-p.Input:
					err := repo.SetDeleted(p.ctx, item)
					if err != nil {
						return err
					}
				case <-p.ctx.Done():
					//log.Println("deleter worker has stopped")
					return nil
				}
			}
		})
	}

	if err := p.g.Wait(); err != nil {
		p.ErrCh <- fmt.Errorf("error in deleter pool: %w", err)
	}
}

func (p DeleterPoolT) Close() {
	_ = p.g.Wait()
	log.Println("deleter pool has closed")
}
