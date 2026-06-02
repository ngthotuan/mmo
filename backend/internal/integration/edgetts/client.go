package edgetts

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

// MergeSRTFiles concatenates per-scene SRT files into one global SRT, shifting
// each file's timestamps by its offset (seconds) so cues line up with the
// concatenated narration. Empty paths are skipped (offset slot still consumed).
func MergeSRTFiles(srtPaths []string, offsetsSec []float64, outPath string) error {
	var b strings.Builder
	counter := 1
	for i, p := range srtPaths {
		if p == "" {
			continue
		}
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		off := 0.0
		if i < len(offsetsSec) {
			off = offsetsSec[i]
		}
		lines := splitLines(string(data))
		j := 0
		for j < len(lines) {
			if strings.Contains(lines[j], "-->") {
				ts := shiftSRTTimestampLine(lines[j], off)
				j++
				var text []string
				for j < len(lines) && strings.TrimSpace(lines[j]) != "" {
					text = append(text, lines[j])
					j++
				}
				b.WriteString(fmt.Sprintf("%d\n%s\n%s\n\n", counter, ts, strings.Join(text, "\n")))
				counter++
			} else {
				j++
			}
		}
	}
	return os.WriteFile(outPath, []byte(b.String()), 0644)
}

func shiftSRTTimestampLine(line string, offsetSec float64) string {
	parts := strings.Split(line, "-->")
	if len(parts) != 2 {
		return line
	}
	return shiftTS(strings.TrimSpace(parts[0]), offsetSec) + " --> " + shiftTS(strings.TrimSpace(parts[1]), offsetSec)
}

func shiftTS(ts string, offsetSec float64) string {
	var h, m, s, ms int
	if _, err := fmt.Sscanf(ts, "%d:%d:%d,%d", &h, &m, &s, &ms); err != nil {
		return ts
	}
	total := (h*3600+m*60+s)*1000 + ms + int(offsetSec*1000+0.5)
	if total < 0 {
		total = 0
	}
	hh := total / 3600000
	total %= 3600000
	mm := total / 60000
	total %= 60000
	ss := total / 1000
	mmm := total % 1000
	return fmt.Sprintf("%02d:%02d:%02d,%03d", hh, mm, ss, mmm)
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
