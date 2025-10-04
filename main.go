package main

import (
	"encoding/json"
	"fmt"
	"newspicker/crawl"
	"newspicker/model"
	"newspicker/twitter"
	"os"

	"github.com/joho/godotenv"
)

const queueFile = "queue.json"

func LoadQueue(filename string) ([]model.ProfitItem, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var queue []model.ProfitItem
	err = json.Unmarshal(data, &queue)
	return queue, err
}

func SaveQueue(filename string, queue []model.ProfitItem) error {
	data, err := json.MarshalIndent(queue, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0644)
}

func main() {
	_ = godotenv.Load()

	queue, err := LoadQueue(queueFile)
	if err != nil || len(queue) == 0 {
		queue, err = crawl.Crawling(12)
		if err != nil {
			fmt.Printf("Queue 저장 실패: %v", err)
			return
		}
	}

	item := queue[0]

	if err := twitter.Upload(item); err != nil {
		fmt.Printf("업로드 실패: %v", err)
	}

	queue = queue[1:]
	if err := SaveQueue(queueFile, queue); err != nil {
		fmt.Printf("Queue 저장 실패: %v", err)
	}
}
