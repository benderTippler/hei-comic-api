package chromedp

import (
	"context"
	"fmt"
	"github.com/chromedp/chromedp"
)

var (
	allocCtx context.Context
	cancel   context.CancelFunc
)

func NewBrowser() (context.Context, context.CancelFunc) {
	if allocCtx == nil {
		allocOpts := append(chromedp.DefaultExecAllocatorOptions[:],
			chromedp.Flag("headless", true),
			chromedp.Flag("window-size", fmt.Sprintf("%d,%d", 80, 80)),
			chromedp.Flag(`disable-extensions`, false),
			chromedp.Flag("blink-settings", "imagesEnabled=false"),
			chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/79.0.3945.79 Safari/537.36"),
		)
		allocCtx, cancel = chromedp.NewExecAllocator(context.Background(), allocOpts...)
	}
	return allocCtx, cancel
}
