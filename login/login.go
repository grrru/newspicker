package login

import (
	"context"
	"log"
	"time"

	"github.com/chromedp/chromedp"
)

func Login(ctx context.Context, id, pw, url string) {
	var ok bool
	if err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.WaitVisible(`input[name="id"]`, chromedp.ByQuery),
		chromedp.SendKeys(`input[name="id"]`, id, chromedp.ByQuery),
		chromedp.SendKeys(`input[name="password"]`, pw, chromedp.ByQuery),
		chromedp.Click(`.btn-confirm`, chromedp.ByQuery),
		chromedp.Sleep(3*time.Second), // 로그인 대기
		chromedp.EvaluateAsDevTools(`!!document.querySelector("section.section01")`, &ok),
	); err != nil {
		log.Fatalln("Login Fail")
		return
	}

	if ok {
		log.Println("LOGIN SUCCESS")
	} else {
		log.Fatalln("After login Fail")
	}
}
