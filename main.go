package main

import (
    "fmt"
    "os"
    "os/exec"
    "log"
    "github.com/go-telegram-bot-api/telegram-bot-api"
)

const (
    botToken = "7195693914:AAH3BE4Mf_XPgvGp8xaj13t85p03sMW3qXM" // Replace with your bot token
)

func runCommand(name string, args ...string) error {
    cmd := exec.Command(name, args...)
    output, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf("command failed: %v\nOutput: %s", err, output)
    }
    fmt.Printf("Output:\n%s\n", output)
    return nil
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
        if update.Message == nil { // ignore non-message updates
            continue
        }

        if update.Message.IsCommand() {
            switch update.Message.Command() {
            case "start":
                msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Send me a YouTube link to download and convert.")
                bot.Send(msg)
            }
            continue
        }

        // Process YouTube link
        if update.Message.Text != "" {
            url := update.Message.Text
            videoFile := "video.mp4.webm"
            audioFile := "audio.mp3"

            // Download video
            if err := runCommand("yt-dlp", "-o", videoFile, url); err != nil {
                log.Println("Error downloading video:", err)
                msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Failed to download the video.")
                bot.Send(msg)
                continue
            } else {
				log.Println("Video download complete")
			}

            // Convert video to audio
            if err := runCommand("ffmpeg", "-i", videoFile, "-q:a", "0", "-map", "a", audioFile); err != nil {
                log.Println("Error converting audio:", err)
                msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Failed to convert the video to audio.")
                bot.Send(msg)
                continue
            } else {
				log.Println("Conversion to audio complete")
			}

            // Check if file exists
            if _, err := os.Stat(audioFile); os.IsNotExist(err) {
                log.Println("Error: Audio file does not exist.")
                msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Audio file not found.")
                bot.Send(msg)
                continue
            } else {
				log.Println("Audio file found")
			}

            audioFileUpload := tgbotapi.NewAudioUpload(update.Message.Chat.ID, audioFile)
            _, err = bot.Send(audioFileUpload)
            if err != nil {
                log.Println("Error sending audio file:", err)
                msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Failed to send the audio file.")
                bot.Send(msg)
                continue
            } else {
				log.Println("Audio sent")
			}

            // Cleanup files
            os.Remove(videoFile)
            os.Remove(audioFile)
        }
    }
}
