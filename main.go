package main

import (
	"context"
	"log"
	"newspicker/login"
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

	NEWSPICK_ID := os.Getenv("NEWSPICK_ID")
	NEWSPICK_PW := os.Getenv("NEWSPICK_PW")
	if NEWSPICK_ID == "" || NEWSPICK_PW == "" {
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

	login.Login(ctx, NEWSPICK_ID, NEWSPICK_PW, NEWSPICK_URL)
}
