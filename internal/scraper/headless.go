package scraper

import (
	"context"
	"fmt"
	"time"

	"github.com/chromedp/chromedp"
	"go.uber.org/zap"
)

func RenderWithChrome(url string, timeout time.Duration) (string, error) {
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, timeout)
	defer cancel()

	var html string
	err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.WaitReady("body", chromedp.ByQuery),
		chromedp.OuterHTML("html", &html),
	)
	if err != nil {
		return "", fmt.Errorf("chromedp render failed for %s: %w", url, err)
	}

	zap.L().Debug("Chrome rendered page", zap.String("url", url), zap.Int("html_len", len(html)))
	return html, nil
}

func RenderWithChromeAndWait(url string, waitForSelector string, timeout time.Duration) (string, error) {
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, timeout)
	defer cancel()

	var html string
	err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.WaitVisible(waitForSelector, chromedp.ByQuery),
		chromedp.OuterHTML("html", &html),
	)
	if err != nil {
		return "", fmt.Errorf("chromedp render failed for %s: %w", url, err)
	}

	return html, nil
}