package core

import (
	"app/graph/model"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/kkdai/youtube/v2"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"gorm.io/gorm"
)

func DownloadAudioFromYouTube(db *gorm.DB, ctx context.Context, songID int, videoURL string) (string, error) {
	// 1. Initialize the specialized scraping client
	client := youtube.Client{}

	// 2. Fetch video streaming blueprints
	video, err := client.GetVideoContext(ctx, videoURL)
	if err != nil {
		log.Printf("Failed to fetch video streams: %v", err)
		return "", fmt.Errorf("fetch video streams: %w", err)
	}

	// 3. Filter list to select ONLY audio tracks (ignoring the visual data track)
	audioFormats := video.Formats.Type("audio")
	if len(audioFormats) == 0 {
		return "", fmt.Errorf("no audio format profiles found for this video")
	}

	// Select the absolute best quality audio track available (typically index 0)
	targetFormat := &audioFormats[0]

	// 4. Request the stream buffer reader
	stream, _, err := client.GetStreamContext(ctx, video, targetFormat)
	if err != nil {
		log.Printf("Failed to open audio data stream: %v", err)
		return "", fmt.Errorf("open audio stream: %w", err)
	}
	defer stream.Close()

	// 5. Build safe target folders and output files
	destinationDir := "/app/downloads"
	if err := os.MkdirAll(destinationDir, 0755); err != nil {
		return "", fmt.Errorf("create download path layout: %w", err)
	}

	// Create an absolute filename (e.g., /downloads/video_id.m4a or .webm)
	outputExtension := "mp3" // Standard default streaming format chunk container
	if targetFormat.MimeType != "" && (len(targetFormat.MimeType) >= 10) {
		// Try to match the extension container type dynamically
		if targetFormat.MimeType[6:10] == "mp3" {
			outputExtension = "mp3"
		}
	}

	outputPath := filepath.Join(destinationDir, fmt.Sprintf("%s.%s", video.ID, outputExtension))
	outFile, err := os.Create(outputPath)

	if err != nil {
		return "", fmt.Errorf("create audio target output file: %w", err)
	}
	defer outFile.Close()

	// 6. Pipe raw incoming network stream buffers onto your hard drive
	log.Printf("Downloading audio for: %s -> %s", video.Title, outputPath)
	_, err = io.Copy(outFile, stream)
	if err != nil {
		return "", fmt.Errorf("failed during data packet streaming pipeline: %w", err)
	}
	fmt.Println("Audio download completed successfully!")
	downloadedRecordURL := fmt.Sprintf("/app/downloads/%s.%s", video.ID, outputExtension)
	fmt.Println(downloadedRecordURL)
	if err := db.Model(&model.Songs{}).Where("song_id = ?", songID).Update("local_link", downloadedRecordURL).Error; err != nil {
		return "", fmt.Errorf("failed to update song link: %w", err)
	}

	if err := db.Model(&model.Song{}).Where("song_id= ?", songID).Update("video_link", videoURL).Update("local_link", downloadedRecordURL).Error; err != nil {
		return "", fmt.Errorf("failed to update song local link: %w", err)
	}
	//transcribeMp3ToText(db, outputPath, songID)

	log.Println("Audio track saved cleanly!")
	return downloadedRecordURL, nil
}

func transcribeMp3ToText(db *gorm.DB, filePath string, song_id int) (string, error) {
	// Initialize official client
	client := openai.NewClient(
		option.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
	)
	log.Println(os.Getenv("OPENAI_API_KEY"))
	log.Println("Getting text from OpenAI")

	// 1. Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open audio file: %w", err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return "", fmt.Errorf("failed to stat audio file: %w", err)
	}
	if info.Size() == 0 {
		return "", fmt.Errorf("audio file is empty")
	}
	log.Printf("Opened audio file %s with size %d bytes", filePath, info.Size())
	// 2. Perform transcription (Pass values directly without wrappers)
	resp, err := client.Audio.Transcriptions.New(context.Background(), openai.AudioTranscriptionNewParams{
		Model: openai.AudioModelWhisper1, // Or simply "whisper-1"
		File:  file,                      // The SDK accepts os.File directly here
	})
	log.Printf("resp = %#v", resp)
	log.Printf("err = %#v", err)
	// CRITICAL: Check the error BEFORE reading or logging 'resp'
	if err != nil {
		return "", fmt.Errorf("OpenAI API error: %w", err)
	}

	log.Println("OpenAI response received successfully")

	// 3. Save to database
	downloadRecord := model.Downloads{
		ID:   song_id,
		File: filePath,
		Text: resp.Text,
	}
	if err := db.Create(&downloadRecord).Error; err != nil {
		return "", fmt.Errorf("failed to save transcription record: %w", err)
	}

	log.Printf("Transcription completed for %s", filePath)
	return resp.Text, nil
}

// func transcribeMp3ToText(db *gorm.DB, filePath string) (string, error) {
// 	// 1. Initialize the Whisper model (ensure you have the correct model file path)
// 	model, err := whisper.New("./models/ggml-base.bin")
// 	if err != nil {
// 		return "", fmt.Errorf("failed to initialize whisper model: %w", err)
// 	}
// 	model.SetNumThreads(6) // Adjust based on your CPU cores
// 	model.SetLanguage("en") // Set the language to English

// 	// 2. Load the audio file
// 	audioData, err := os.ReadFile(filePath)
// 	if err != nil {
// 		return "", fmt.Errorf("failed to read audio file: %w", err)
// 	}

// 	// 3. Perform transcription
// 	result, err := model.Transcribe(audioData)
// 	if err != nil {
// 		return "", fmt.Errorf("failed to transcribe audio: %w", err)
// 	}
// 	downloadRecord := model.Download{
// 		"FilePath": filePath,
// 		"Text":     result.Text,
// 	}
// 	db.Create(&downloadRecord)
// 	log.Printf("Transcription completed for %s: %s", filePath, result.Text)
// 	return result.Text, nil
// }
