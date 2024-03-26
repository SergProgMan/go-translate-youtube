package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
)

type Config struct {
	DeeplApiKey    string `json:"deepl_api_key"`
	YoutubeApiKey  string `json:"youtube_api_key"`
	YoutubeVideoId string `json:"youtube_video_id"`
}

type TranslationResponse struct {
	Translations []struct {
		DetectedSourceLanguage string `json:"detected_source_language"`
		Text                   string `json:"text"`
	} `json:"translations"`
}

type DeeplLanguage struct {
	Code string `json:"language"`
	Name string `json:"name"`
}

type DeeplLanguagesResponse struct {
	LanguageList []struct {
		Code string `json:"language"`
		Name string `json:"name"`
	} `json:"languages"`
}

type YouTubeVideo struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

func loadConfig(filename string) (Config, error) {
	var config Config

	configFile, err := os.ReadFile(filename)
	if err != nil {
		return config, err
	}

	err = json.Unmarshal(configFile, &config)
	if err != nil {
		return config, err
	}

	return config, nil
}

func fetchYouTubeVideoInfo(videoID string, apiKey string) (YouTubeVideo, error) {
	url := fmt.Sprintf("https://www.googleapis.com/youtube/v3/videos?id=%s&key=%s&part=snippet", videoID, apiKey)

	resp, err := http.Get(url)
	if err != nil {
		return YouTubeVideo{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return YouTubeVideo{}, fmt.Errorf("failed to fetch video information, status code: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return YouTubeVideo{}, err
	}

	var response struct {
		Items []struct {
			Snippet struct {
				Title       string `json:"title"`
				Description string `json:"description"`
			} `json:"snippet"`
		} `json:"items"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return YouTubeVideo{}, err
	}

	if len(response.Items) == 0 {
		return YouTubeVideo{}, fmt.Errorf("video with ID %s not found", videoID)
	}

	return YouTubeVideo{
		ID:          videoID,
		Title:       response.Items[0].Snippet.Title,
		Description: response.Items[0].Snippet.Description,
	}, nil
}

func getDeeplLanguages(apiKey string) ([]DeeplLanguage, error) {
	url := "https://api-free.deepl.com/v2/languages"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "DeepL-Auth-Key "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var languages []DeeplLanguage
	if err := json.NewDecoder(resp.Body).Decode(&languages); err != nil {
		return nil, err
	}

	return languages, nil
}

func translateText(text string, apiKey string, targetLang string) (string, error) {
	url := "https://api-free.deepl.com/v2/translate"

	// Prepare translation request
	data := map[string]interface{}{
		"text":        []string{text},
		"target_lang": targetLang,
	}
	requestData, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request data: %v", err)
	}

	// Send request to DeepL API
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestData))
	if err != nil {
		return "", fmt.Errorf("failed to create HTTP request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "DeepL-Auth-Key "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send HTTP request: %v", err)
	}
	defer resp.Body.Close()

	// Check HTTP response status code
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP request failed with status code: %d", resp.StatusCode)
	}

	// Parse response
	var translationResponse TranslationResponse
	if err := json.NewDecoder(resp.Body).Decode(&translationResponse); err != nil {
		return "", fmt.Errorf("failed to parse response body: %v", err)
	}

	// Check if translations are available
	if len(translationResponse.Translations) == 0 {
		return "", errors.New("no translations found")
	}

	// Extract translated text
	translatedText := translationResponse.Translations[0].Text

	return translatedText, nil
}

func main() {
	fmt.Println("Starting...")
	config, err := loadConfig("config.json")
	if err != nil {
		fmt.Println("Error loading config:", err)
		return
	}

	apiKey := config.DeeplApiKey
	fmt.Println("apiKey:", apiKey)
	// textToTranslate := "Hello, world!"

	// translatedText, err := translateText(textToTranslate, apiKey, "DE")
	// if err != nil {
	// 	fmt.Println("Error:", err)
	// 	return
	// }

	// fmt.Println("Translated text:", translatedText)

	deepLLanguages, err := getDeeplLanguages(apiKey)
	if err != nil {
		fmt.Println("Error fetching DeepL languages:", err)
		return
	}

	fmt.Println("DeepL Supported Languages:")
	for _, lang := range deepLLanguages {
		fmt.Printf("Code: %s, Name: %s\n", lang.Code, lang.Name)
	}

	youtubeApiKey := config.YoutubeApiKey
	videoID := config.YoutubeVideoId
	videoInfo, err := fetchYouTubeVideoInfo(videoID, youtubeApiKey)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println("Title:", videoInfo.Title)
	fmt.Println("Description:", videoInfo.Description)
}
