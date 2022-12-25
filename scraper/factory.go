package scraper

import (
	"context"
	"github.com/chromedp/chromedp"
	"github.com/m4schini/logger"
	"golang.org/x/time/rate"
)

var limiter = rate.NewLimiter(rate.Limit(1), 1)

type AllocatorContextFactory struct {
	execAllocatorCtx   context.Context
	cancelAllocatorCtx context.CancelFunc
}

func NewAllocatorContextFactory(ctx context.Context, proxyAddr string) *AllocatorContextFactory {
	af := new(AllocatorContextFactory)
	opts := []chromedp.ExecAllocatorOption{
		chromedp.Flag("-incognito", true),
	}

	if !logger.DevelopmentMode {
		for _, option := range chromedp.DefaultExecAllocatorOptions {
			opts = append(opts, option)
		}
	}

	if proxyAddr != "" {
		opts = append(opts, chromedp.ProxyServer(proxyAddr))
	}

	af.execAllocatorCtx, af.cancelAllocatorCtx = chromedp.NewExecAllocator(ctx, opts...)
	return af
}

func (a *AllocatorContextFactory) NewContext() (context.Context, context.CancelFunc) {
	limiter.Wait(context.Background())
	return chromedp.NewContext(a.execAllocatorCtx)
}

func (a *AllocatorContextFactory) Close() error {
	a.cancelAllocatorCtx()
	return nil
}
