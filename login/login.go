package login

import (
	"context"
	"errors"
	"time"

	"github.com/chromedp/chromedp"
)

func DoLogin(ctx context.Context, id, pw, url string) error {
	if err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.WaitVisible(`input[name="id"]`, chromedp.ByQuery),
		chromedp.SendKeys(`input[name="id"]`, id, chromedp.ByQuery),
		chromedp.SendKeys(`input[name="password"]`, pw, chromedp.ByQuery),
		chromedp.Click(`.btn-confirm`, chromedp.ByQuery),
	); err != nil {
		return errors.New("login form submit failed")
	}

	wctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	selMyPage := `a[href="/management/operation/myPage"]`
	if err := chromedp.Run(wctx,
		chromedp.WaitVisible(selMyPage, chromedp.ByQuery),
	); err != nil {
		return errors.New("login timeout: my page not visible")
	}

	return nil
}
