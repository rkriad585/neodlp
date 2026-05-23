package uploader

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"neodlp/internal/config"
)

// Upload triggers the configured upload method for a given local file path.
// It returns an error if the upload fails.
func Upload(target string, filePath string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration for upload: %w", err)
	}

	absPath, err := filepath.Abs(filePath)
	if err != nil {
		absPath = filePath
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return fmt.Errorf("failed to access file %s: %w", filePath, err)
	}
	if info.IsDir() {
		return fmt.Errorf("cannot upload directory: %s", absPath)
	}

	fmt.Printf("  ☁️ Initiating cloud upload to %s for: %s\n", target, filepath.Base(absPath))

	switch strings.ToLower(target) {
	case "telegram":
		err = uploadTelegram(cfg.Upload.Telegram, absPath)
	case "discord":
		err = uploadDiscord(cfg.Upload.Discord, absPath)
	case "custom":
		err = uploadCustom(cfg.Upload.Custom, absPath)
	default:
		return fmt.Errorf("unsupported upload target: %s", target)
	}

	if err != nil {
		return fmt.Errorf("upload to %s failed: %w", target, err)
	}

	fmt.Printf("  ✓ Success! File uploaded successfully to %s.\n", target)

	// Clean up local file if successfully uploaded
	fmt.Printf("  🧹 Cleaning up local file: %s\n", filepath.Base(absPath))
	if err := os.Remove(absPath); err != nil {
		fmt.Fprintf(os.Stderr, "  ! Warning: failed to delete local file: %v\n", err)
	}

	return nil
}

// uploadTelegram posts the file as a document via Telegram Bot API
func uploadTelegram(cfg config.TelegramUploadConfig, filePath string) error {
	if cfg.BotToken == "" || cfg.ChatID == "" {
		return fmt.Errorf("telegram bot_token or chat_id is missing in config.toml")
	}

	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add chat_id parameter
	if err := writer.WriteField("chat_id", cfg.ChatID); err != nil {
		return err
	}

	// Add file parameter
	part, err := writer.CreateFormFile("document", filepath.Base(filePath))
	if err != nil {
		return err
	}
	if _, err := io.Copy(part, file); err != nil {
		return err
	}

	if err := writer.Close(); err != nil {
		return err
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendDocument", cfg.BotToken)
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{Timeout: 30 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram server returned status %s: %s", resp.Status, string(respBody))
	}

	return nil
}

// uploadDiscord posts the file via Discord Webhook multipart attachment
func uploadDiscord(cfg config.DiscordUploadConfig, filePath string) error {
	if cfg.WebhookURL == "" {
		return fmt.Errorf("discord webhook_url is missing in config.toml")
	}

	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("files[0]", filepath.Base(filePath))
	if err != nil {
		return err
	}
	if _, err := io.Copy(part, file); err != nil {
		return err
	}

	if err := writer.Close(); err != nil {
		return err
	}

	req, err := http.NewRequest("POST", cfg.WebhookURL, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{Timeout: 30 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("discord server returned status %s: %s", resp.Status, string(respBody))
	}

	return nil
}

// uploadCustom executes a user-defined shell command, replacing placeholders
func uploadCustom(cfg config.CustomUploadConfig, filePath string) error {
	if cfg.Command == "" {
		return fmt.Errorf("custom upload command is missing in config.toml")
	}

	absPath, _ := filepath.Abs(filePath)
	dir := filepath.Dir(absPath)
	filename := filepath.Base(absPath)

	// Replace placeholders
	cmdStr := cfg.Command
	cmdStr = strings.ReplaceAll(cmdStr, "%file%", absPath)
	cmdStr = strings.ReplaceAll(cmdStr, "%filename%", filename)
	cmdStr = strings.ReplaceAll(cmdStr, "%dir%", dir)

	fmt.Printf("  [shell] Executing: %s\n", cmdStr)

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", cmdStr)
	} else {
		cmd = exec.Command("sh", "-c", cmdStr)
	}

	// Capture output for diagnosis
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("custom command failed: %v\nstderr: %s\nstdout: %s", err, stderr.String(), stdout.String())
	}

	return nil
}
