package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"github.com/joho/godotenv"
)

const (
	NEWSPICK_URL = "https://partners.newspic.kr/main/index"
)

func main() {
	_ = godotenv.Load()

	id := os.Getenv("NEWSPICK_ID")
	pw := os.Getenv("NEWSPICK_PW")
	if id == "" || pw == "" {
		log.Fatal("환경변수 NEWSPICK_ID/NEWSPICK_PW 가 필요합니다 (.env 파일 확인)")
	}

	headless := os.Getenv("HEADLESS") == "1"

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", headless),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
	)

	alloc, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel2 := chromedp.NewContext(alloc)
	defer cancel2()

	// 전체 타임아웃
	ctx, cancel3 := context.WithTimeout(ctx, 40*time.Second)
	defer cancel3()

	// JS alert/confirm 자동 처리 (chromedp는 이벤트 리스너 방식)
	chromedp.ListenTarget(ctx, func(ev any) {
		if _, ok := ev.(*page.EventJavascriptDialogOpening); ok {
			go func() {
				_ = chromedp.Run(ctx, page.HandleJavaScriptDialog(true))
			}()
		}
	})

	var ok bool
	if err := chromedp.Run(ctx,
		chromedp.Navigate(NEWSPICK_URL),
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
	}
}
