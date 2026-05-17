package edgetts

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"mmo/pkg/config"
)

type Client struct {
	pythonBin    string
	defaultVoice string
}

type GenerateResult struct {
	AudioPath    string
	SubtitlePath string
}

func New(cfg config.EdgeTTSConfig) *Client {
	python := "python3"
	if path, err := exec.LookPath("python3"); err == nil {
		python = path
	}
	return &Client{
		pythonBin:    python,
		defaultVoice: cfg.DefaultVoice,
	}
}

func (c *Client) DefaultVoice() string {
	return c.defaultVoice
}

func (c *Client) Generate(ctx context.Context, text, voice, outputDir string) (*GenerateResult, error) {
	if voice == "" {
		voice = c.defaultVoice
	}

	ts := fmt.Sprintf("%d", time.Now().UnixNano())
	audioPath := filepath.Join(outputDir, "tts_"+ts+".mp3")
	subtitlePath := filepath.Join(outputDir, "tts_"+ts+".vtt")

	cmd := exec.CommandContext(ctx, c.pythonBin, "-m", "edge_tts",
		"--voice", voice,
		"--text", text,
		"--write-media", audioPath,
		"--write-subtitles", subtitlePath,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		cmd2 := exec.CommandContext(ctx, "edge-tts",
			"--voice", voice,
			"--text", text,
			"--write-media", audioPath,
			"--write-subtitles", subtitlePath,
		)
		if err2 := cmd2.Run(); err2 != nil {
			return nil, fmt.Errorf("edge-tts failed: %w (fallback: %v)", err, err2)
		}
	}

	return &GenerateResult{
		AudioPath:    audioPath,
		SubtitlePath: subtitlePath,
	}, nil
}

func VTTToSRT(vttPath string) (string, error) {
	data, err := os.ReadFile(vttPath)
	if err != nil {
		return "", err
	}

	srtPath := vttPath[:len(vttPath)-4] + ".srt"
	lines := splitLines(string(data))
	var srtLines []string
	counter := 1
	i := 0
	for i < len(lines) {
		if lines[i] == "WEBVTT" || lines[i] == "" {
			i++
			continue
		}
		if len(lines[i]) > 10 && lines[i][2] == ':' {
			srtLines = append(srtLines, fmt.Sprintf("%d", counter))
			ts := replaceDotsInTimestamp(lines[i])
			srtLines = append(srtLines, ts)
			counter++
			i++
			for i < len(lines) && lines[i] != "" {
				srtLines = append(srtLines, lines[i])
				i++
			}
			srtLines = append(srtLines, "")
		} else {
			i++
		}
	}

	if err := os.WriteFile(srtPath, []byte(joinLines(srtLines)), 0644); err != nil {
		return "", err
	}
	return srtPath, nil
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			line := s[start:i]
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}
			lines = append(lines, line)
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func joinLines(lines []string) string {
	result := ""
	for _, l := range lines {
		result += l + "\n"
	}
	return result
}

func replaceDotsInTimestamp(ts string) string {
	result := []byte(ts)
	for i := range result {
		if result[i] == '.' && i > 5 {
			result[i] = ','
		}
	}
	return string(result)
}
