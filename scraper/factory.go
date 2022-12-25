package scraper

import (
	"context"
	"github.com/chromedp/chromedp"
	"web-scraper-module/config"
)

func NewChromeContext(ctx context.Context, proxyAddr string) (context.Context, context.CancelFunc) {
	opts := []chromedp.ExecAllocatorOption{
		chromedp.Flag("-incognito", true),
	}

	if !config.DevModeEnabled() {
		for _, option := range chromedp.DefaultExecAllocatorOptions {
			opts = append(opts, option)
		}
	}

	if proxyAddr != "" {
		opts = append(opts, chromedp.ProxyServer(proxyAddr))
	}

	execAllocatorCtx, cancelAllocatorCtx := chromedp.NewExecAllocator(ctx, opts...)
	ctx, cancel := chromedp.NewContext(execAllocatorCtx)
	return ctx, func() {
		cancel()
		cancelAllocatorCtx()
	}
}
