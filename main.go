package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

const (
	botToken = "7195693914:AAH3BE4Mf_XPgvGp8xaj13t85p03sMW3qXM" // Replace with your bot token
)

var userChoices = make(map[int64]struct {
	URL       string
	VideoFile string
	AudioFile string
})

// RunCommand runs a command and returns its output and error
func runCommand(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// ConvertWebMToMP4 converts a .mp4.webm file to .mp4
func ConvertWebMToMP4(inputFile string) (string, error) {
	outputFile := strings.TrimSuffix(inputFile, ".webm") + ".mp4" // Removing ".webm" and adding ".mp4"
	cmd := exec.Command("ffmpeg", "-i", inputFile, outputFile)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("error running ffmpeg command: %v, output: %s", err, string(output))
	}
	return outputFile, nil
}

// HandleTextMessage processes the YouTube link and presents options to the user
func handleTextMessage(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	url := update.Message.Text

	// Get video title
	title, err := getVideoTitle(url)
	if err != nil {
		log.Println("Error getting video title:", err)
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Failed to get video title.")
		bot.Send(msg)
		return
	}

	// Define file names based on title
	videoFile := fmt.Sprintf("%s.mp4.webm", title)
	audioFile := fmt.Sprintf("%s.mp3", title)

	// Create inline keyboard for user to choose between video or audio
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Video", "video"),
			tgbotapi.NewInlineKeyboardButtonData("Audio", "audio"),
		),
	)

	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Choose the format you want:")
	msg.ReplyMarkup = keyboard
	bot.Send(msg)

	// Store URL, video file, and audio file names for later processing
	userChoices[update.Message.Chat.ID] = struct {
		URL       string
		VideoFile string
		AudioFile string
	}{
		URL:       url,
		VideoFile: videoFile,
		AudioFile: audioFile,
	}
}

// HandleCallbackQuery processes the user's choice (video or audio)
func handleCallbackQuery(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	chatID := update.CallbackQuery.Message.Chat.ID
	choice := update.CallbackQuery.Data

	userChoice, exists := userChoices[chatID]
	if !exists {
		msg := tgbotapi.NewMessage(chatID, "No YouTube link found.")
		bot.Send(msg)
		return
	}

	// Download video
	_, err := runCommand("yt-dlp", "-o", userChoice.VideoFile, userChoice.URL)
	if err != nil {
		log.Println("Error downloading video:", err)
		msg := tgbotapi.NewMessage(chatID, "Failed to download the video.")
		bot.Send(msg)
		return
	}

	// Convert video to .mp4 if needed
	if choice == "video" {
		videoFileMP4, err := ConvertWebMToMP4(userChoice.VideoFile)
		if err != nil {
			log.Println("Error converting video:", err)
			msg := tgbotapi.NewMessage(chatID, "Failed to convert the video.")
			bot.Send(msg)
			return
		}

		// Send the video file
		videoToSend := tgbotapi.NewVideoUpload(chatID, videoFileMP4)
		if _, err := bot.Send(videoToSend); err != nil {
			log.Println("Error sending video file:", err)
			msg := tgbotapi.NewMessage(chatID, "Failed to send the video file.")
			bot.Send(msg)
			return
		}

		// Cleanup files
		os.Remove(userChoice.VideoFile)
		os.Remove(videoFileMP4)
	} else if choice == "audio" {
		// Convert video to audio
		_, err := runCommand("ffmpeg", "-i", userChoice.VideoFile, "-q:a", "0", "-map", "a", userChoice.AudioFile)
		if err != nil {
			log.Println("Error converting audio:", err)
			msg := tgbotapi.NewMessage(chatID, "Failed to convert the video to audio.")
			bot.Send(msg)
			return
		}

		// Send the audio file
		audioFileToSend := tgbotapi.NewAudioUpload(chatID, userChoice.AudioFile)
		if _, err := bot.Send(audioFileToSend); err != nil {
			log.Println("Error sending audio file:", err)
			msg := tgbotapi.NewMessage(chatID, "Failed to send the audio file.")
			bot.Send(msg)
			return
		}

		// Cleanup files
		os.Remove(userChoice.VideoFile)
		os.Remove(userChoice.AudioFile)
	}

	// Remove URL from user choices
	delete(userChoices, chatID)
}

// GetVideoTitle retrieves the title of the video from the URL
func getVideoTitle(url string) (string, error) {
	output, err := runCommand("yt-dlp", "--get-title", url)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(output), nil
}

func main() {
	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)
	if err != nil {
		log.Fatal(err)
	}

	for update := range updates {
		if update.Message != nil {
			if update.Message.IsCommand() {
				switch update.Message.Command() {
				case "start":
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Send me a YouTube link to download.")
					bot.Send(msg)
				}
			} else if update.Message.Text != "" {
				handleTextMessage(bot, update)
			}
		} else if update.CallbackQuery != nil {
			handleCallbackQuery(bot, update)
		}
	}
}
