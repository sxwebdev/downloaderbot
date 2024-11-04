package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	"github.com/PuerkitoBio/goquery"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "Instagram Reel Downloader",
		Usage: "Download Instagram Reels",
		Action: func(c *cli.Context) error {
			// Укажите ваш логин и пароль от Instagram
			username := "your_username"
			password := "your_password"

			// Ссылка на рилс
			reelURL := "https://www.instagram.com/reel/CzBjgFiISfF/?igshid=MzRlODBiNWFlZA=="

			client, err := loginInstagram(username, password)
			if err != nil {
				return fmt.Errorf("login failed: %v", err)
			}

			videoURL, err := extractReelVideoURL(client, reelURL)
			if err != nil {
				return fmt.Errorf("failed to extract video URL: %v", err)
			}

			err = downloadVideo(client, videoURL, "reel_video.mp4")
			if err != nil {
				return fmt.Errorf("failed to download video: %v", err)
			}

			fmt.Println("Video downloaded successfully as reel_video.mp4")
			return nil
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}

// loginInstagram выполняет авторизацию на Instagram и возвращает HTTP-клиент с сохраненными cookies
func loginInstagram(username, password string) (*http.Client, error) {
	client := &http.Client{}

	data := url.Values{}
	data.Set("username", username)
	data.Set("enc_password", fmt.Sprintf("#PWD_INSTAGRAM_BROWSER:0:%d:%s", 0, password)) // Обратите внимание на формат пароля

	req, err := http.NewRequest("POST", "https://www.instagram.com/accounts/login/ajax/", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-CSRFToken", "your-csrf-token") // Вам нужно будет найти и использовать правильный CSRF токен
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Referer", "https://www.instagram.com/")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/85.0.4183.121 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to login, status code: %d", resp.StatusCode)
	}

	return client, nil
}

// extractReelVideoURL парсит страницу рилса и извлекает URL видео
func extractReelVideoURL(client *http.Client, reelURL string) (string, error) {
	req, err := http.NewRequest("GET", reelURL, nil)
	if err != nil {
		return "", err
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", err
	}

	var videoURL string
	doc.Find("meta[property='og:video']").Each(func(i int, s *goquery.Selection) {
		videoURL, _ = s.Attr("content")
	})

	if videoURL == "" {
		return "", fmt.Errorf("video URL not found")
	}

	return videoURL, nil
}

// downloadVideo скачивает видео и сохраняет его на диск
func downloadVideo(client *http.Client, videoURL, filename string) error {
	resp, err := client.Get(videoURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	return err
}
