package twitter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"newspicker/model"
	"os"

	"github.com/dghubble/oauth1"
)

func Upload(item model.ProfitItem) error {
	apiKey := os.Getenv("TWITTER_API_KEY")
	apiSecret := os.Getenv("TWITTER_API_SECRET")
	accessToken := os.Getenv("TWITTER_ACCESS_TOKEN")
	accessTokenSecret := os.Getenv("TWITTER_ACCESS_TOKEN_SECRET")

	if apiKey == "" || apiSecret == "" || accessToken == "" || accessTokenSecret == "" {
		return fmt.Errorf("twitter 환경변수 누락")
	}

	config := oauth1.NewConfig(apiKey, apiSecret)
	token := oauth1.NewToken(accessToken, accessTokenSecret)
	httpClient := config.Client(oauth1.NoContext, token)

	tweetText := fmt.Sprintf("%s\n\n%s", item.Title, item.Link)
	payload := map[string]string{
		"text": tweetText,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("JSON 변환 실패: %v", err)
	}

	req, err := http.NewRequest("POST", "https://api.twitter.com/2/tweets", bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("HTTP 요청 생성 실패: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// OAuth 1.0a 인증된 클라이언트로 요청 실행
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP 요청 실행 실패: %v", err)
	}
	defer resp.Body.Close()

	// 응답 읽기
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("응답 읽기 실패: %v", err)
	}

	// 응답 상태 확인
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("트윗 업로드 실패 - 상태코드: %s | 응답: %s", resp.Status, string(respBody))
	}

	fmt.Printf("트윗 업로드 성공: %s\n", item.Title)
	fmt.Printf("응답 상태: %s\n", resp.Status)

	return nil
}
