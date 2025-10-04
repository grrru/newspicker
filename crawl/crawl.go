package crawl

import (
	"context"
	"fmt"
	"log"
	"newspicker/model"
	"os"
	"strings"
	"time"

	"github.com/chromedp/cdproto/browser"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/cdproto/input"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
)

const (
	NEWSPICK_URL = "https://partners.newspic.kr/main/index"
)

func Crawling(cnt int) ([]model.ProfitItem, error) {
	NEWSPICK_ID := os.Getenv("NEWSPICK_ID")
	NEWSPICK_PW := os.Getenv("NEWSPICK_PW")
	if NEWSPICK_ID == "" || NEWSPICK_PW == "" {
		log.Fatal("need NEWSPICK_ID/NEWSPICK_PW (check .env)")
	}

	headless := os.Getenv("HEADLESS") != "0"

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", headless),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-background-timer-throttling", true),
		chromedp.Flag("disable-backgrounding-occluded-windows", true),
		chromedp.Flag("disable-renderer-backgrounding", true),
		chromedp.Flag("disable-features", "TranslateUI"),
		chromedp.Flag("disable-ipc-flooding-protection", true),
		chromedp.Flag("disable-web-security", true),
		chromedp.Flag("disable-extensions", true),
		chromedp.Flag("disable-plugins", true),
		chromedp.Flag("disable-default-apps", true),
		chromedp.Flag("disable-sync", true),
		chromedp.Flag("no-first-run", true),
		chromedp.Flag("no-default-browser-check", true),
		chromedp.Flag("remote-debugging-port", "9222"),
		chromedp.WindowSize(1920, 1080),
	)

	alloc, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel2 := chromedp.NewContext(alloc)
	defer cancel2()

	// 전체 타임아웃
	ctx, cancel3 := context.WithTimeout(ctx, 40*time.Second)
	defer cancel3()

	// ★ 반드시 가장 먼저 활성화
	if err := chromedp.Run(ctx, page.Enable()); err != nil {
		log.Fatal(err)
	}

	// JS 다이얼로그 무시
	_ = chromedp.Run(ctx, chromedp.ActionFunc(func(ctx context.Context) error {
		_, err := page.AddScriptToEvaluateOnNewDocument(`
		  (function(){
			if (window.__np_patch) return;
			window.__np_patch = 1;
			window.alert = function(){};
			window.confirm = function(){ return true; };
		  })();
		`).Do(ctx)
		return err
	}))

	// 클립보드 읽기쓰기 권한 허용
	chromedp.Run(ctx,
		browser.GrantPermissions(
			[]browser.PermissionType{
				browser.PermissionTypeClipboardReadWrite,
				browser.PermissionTypeClipboardSanitizedWrite,
			},
		).WithOrigin("https://partners.newspic.kr"),
	)

	if err := DoLogin(ctx, NEWSPICK_ID, NEWSPICK_PW, NEWSPICK_URL); err != nil {
		log.Fatal(err)
		return nil, err
	}
	profitItems, err := ScrapeProfitNews(ctx, cnt)
	if err != nil {
		return nil, fmt.Errorf("크롤링 실패: %v", err)
	}

	return profitItems, nil
}

func ScrapeProfitNews(ctx context.Context, max int) ([]model.ProfitItem, error) {
	// 섹션 로딩 대기
	if err := chromedp.Run(ctx,
		chromedp.WaitVisible(`section.section01`, chromedp.ByQuery),
	); err != nil {
		return nil, fmt.Errorf("section01 not visible: %w", err)
	}

	// li 카드 수집
	var lis []*cdp.Node
	if err := chromedp.Run(ctx,
		chromedp.Nodes(`section.section01 li.swiper-slide`, &lis, chromedp.ByQueryAll),
	); err != nil {
		return nil, fmt.Errorf("failed to query list items: %w", err)
	}

	if len(lis) == 0 {
		return nil, fmt.Errorf("no items found (check selectors or login)")
	}

	if max > 0 && max < len(lis) {
		lis = lis[:max]
	}

	items := make([]model.ProfitItem, 0, len(lis))

	for _, li := range lis {
		var title, img string

		err := chromedp.Run(ctx, chromedp.Tasks{
			chromedp.ScrollIntoView(`div.thumb`, chromedp.FromNode(li), chromedp.ByQuery),
			chromedp.Text(`span.text-overflow2`, &title, chromedp.FromNode(li), chromedp.ByQuery, chromedp.NodeVisible),
			chromedp.AttributeValue(`img[alt="기사 대표이미지"]`, "src", &img, nil, chromedp.FromNode(li), chromedp.ByQuery),
		})
		if err != nil {
			log.Printf("extract error on one item: %v", err)
			continue
		}

		// 제목 정리
		title = strings.ReplaceAll(title, " …", "")
		title = strings.ReplaceAll(title, `'`, " ")
		title = strings.ReplaceAll(title, `"`, " ")
		title = strings.TrimSpace(title)

		// 복사 버튼 클릭 → 클립보드 읽기
		link, err := clickCopyAndRead(ctx, li)
		if err != nil || link == "" {
			log.Printf("copy/read link failed : %v", err)
		}

		item := model.ProfitItem{Image: img, Title: title, Link: link}
		items = append(items, item)
	}

	if len(items) == 0 {
		return nil, fmt.Errorf("scrape ended but no valid items composed")
	}
	return items, nil
}

func clickCopyAndRead(ctx context.Context, li *cdp.Node) (string, error) {

	if err := click(ctx, li); err != nil {
		return "", fmt.Errorf("click fail: %w", err)
	}

	time.Sleep(200 * time.Millisecond)

	// 3) navigator.clipboard.readText() 평가 (Promise await)
	return clipboard(ctx)
}

func click(ctx context.Context, li *cdp.Node) error {
	const (
		thumbSel = `div.thumb`
		btnSel   = `[data-type="copyurl"]`
	)

	// 1) 썸(이미지) 영역으로 스크롤
	if err := chromedp.Run(ctx,
		chromedp.ScrollIntoView(thumbSel, chromedp.FromNode(li), chromedp.ByQuery),
		chromedp.Sleep(200*time.Millisecond),
	); err != nil {
		return fmt.Errorf("li scroll fail: %w", err)
	}

	// 2) 썸 노드 찾기
	var thumbs []*cdp.Node
	if err := chromedp.Run(ctx,
		chromedp.Nodes(thumbSel, &thumbs, chromedp.FromNode(li), chromedp.ByQuery),
	); err != nil {
		return fmt.Errorf("thumb query fail: %w", err)
	}
	if len(thumbs) == 0 {
		return fmt.Errorf("thumb not found under li")
	}
	thumb := thumbs[0]

	// 3) 썸 중앙 좌표로 마우스 이동(hover 유도)
	if err := chromedp.Run(ctx, chromedp.ActionFunc(func(ctx context.Context) error {
		box, err := dom.GetBoxModel().WithNodeID(thumb.NodeID).Do(ctx)
		if err != nil {
			return fmt.Errorf("get box model fail: %w", err)
		}
		if len(box.Content) < 8 {
			return fmt.Errorf("invalid box model")
		}
		// Content: x0,y0,x1,y1,x2,y2,x3,y3 (사각형 4꼭짓점)
		c := box.Content
		x := (c[0] + c[2]) / 2
		y := (c[1] + c[5]) / 2

		// 실제 마우스 이동 이벤트(hover 트리거)
		return input.DispatchMouseEvent(input.MouseMoved, x, y).Do(ctx)
	}),
		chromedp.Sleep(500*time.Millisecond), // hover로 버튼 노출 대기
	); err != nil {
		return fmt.Errorf("hover move fail: %w", err)
	}

	// 4) 복사 버튼 노출될 때까지 대기
	if err := chromedp.Run(ctx,
		chromedp.WaitVisible(btnSel, chromedp.FromNode(li), chromedp.ByQuery),
	); err != nil {
		return fmt.Errorf("copy button not visible: %w", err)
	}

	// 5) 버튼 노드 찾아서 클릭
	var btns []*cdp.Node
	if err := chromedp.Run(ctx,
		chromedp.Nodes(btnSel, &btns, chromedp.FromNode(li), chromedp.ByQueryAll),
	); err != nil {
		return fmt.Errorf("btn query fail under li: %w", err)
	}
	if len(btns) == 0 {
		return fmt.Errorf("no copy button under li")
	}

	if err := chromedp.Run(ctx, chromedp.MouseClickNode(btns[0])); err != nil {
		return fmt.Errorf("click failed: %w", err)
	}

	return nil
}

func clipboard(ctx context.Context) (string, error) {
	var clip string
	if err := chromedp.Run(ctx, chromedp.ActionFunc(func(ctx context.Context) error {
		// DevTools Evaluate는 Promise를 자동으로 await합니다
		ro, exp, err := runtime.Evaluate(`navigator.clipboard.readText()`).WithAwaitPromise(true).Do(ctx)
		if err != nil {
			return fmt.Errorf("clipboard evaluate error: %w", err)
		}
		if exp != nil {
			return fmt.Errorf("clipboard evaluate exception: %s", exp.Text)
		}
		if ro == nil || ro.Value == nil {
			clip = ""
			return nil
		}
		clip = ro.Value.String()
		return nil
	})); err != nil {
		return "", err
	}

	clip = strings.Trim(clip, `"`)

	return clip, nil
}
