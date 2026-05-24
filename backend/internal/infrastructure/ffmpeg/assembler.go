package ffmpeg

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

type Assembler struct {
	ffmpegBin string
	cfg       config.FFmpegConfig
}

type MediaAsset struct {
	Path     string
	Type     string
	Duration float64
}

type AssembleResult struct {
	VideoPath       string
	DurationSeconds float64
	FileSizeBytes   int64
	Log             string
}

func New(cfg config.FFmpegConfig) *Assembler {
	bin := "ffmpeg"
	if path, err := exec.LookPath("ffmpeg"); err == nil {
		bin = path
	}
	return &Assembler{ffmpegBin: bin, cfg: cfg}
}

func (a *Assembler) TempDir(jobID string) (string, error) {
	base := a.cfg.TempDir
	if base == "" {
		base = filepath.Join(os.TempDir(), "mmo-media")
	}
	dir := filepath.Join(base, jobID)
	return dir, os.MkdirAll(dir, 0755)
}

func (a *Assembler) CleanupTempDir(jobID string) {
	base := a.cfg.TempDir
	if base == "" {
		base = filepath.Join(os.TempDir(), "mmo-media")
	}
	_ = os.RemoveAll(filepath.Join(base, jobID))
}

func (a *Assembler) AssembleSlideshow(ctx context.Context, assets []MediaAsset, audioPath, srtPath, outputDir string) (*AssembleResult, error) {
	if len(assets) == 0 {
		return nil, fmt.Errorf("no assets provided for slideshow")
	}

	ts := fmt.Sprintf("%d", time.Now().UnixNano())
	outputPath := filepath.Join(outputDir, "video_"+ts+".mp4")

	var args []string
	for _, asset := range assets {
		if asset.Type == "image" {
			args = append(args, "-loop", "1", "-t", fmt.Sprintf("%.1f", asset.Duration), "-i", asset.Path)
		} else {
			args = append(args, "-i", asset.Path)
		}
	}
	args = append(args, "-i", audioPath)

	n := len(assets)
	var filterParts []string
	for i := range assets {
		filterParts = append(filterParts, fmt.Sprintf("[%d:v]scale=%d:%d:force_original_aspect_ratio=decrease,pad=%d:%d:(ow-iw)/2:(oh-ih)/2:black,fps=%d[v%d]",
			i, a.cfg.OutputWidth, a.cfg.OutputHeight, a.cfg.OutputWidth, a.cfg.OutputHeight, a.cfg.OutputFPS, i))
	}
	concatInputs := ""
	for i := range assets {
		concatInputs += fmt.Sprintf("[v%d]", i)
	}
	filterParts = append(filterParts, fmt.Sprintf("%sconcat=n=%d:v=1:a=0[outv]", concatInputs, n))
	filterComplex := strings.Join(filterParts, ";")

	args = append(args,
		"-filter_complex", filterComplex,
		"-map", "[outv]",
		"-map", fmt.Sprintf("%d:a", n),
		"-c:v", "libx264",
		"-crf", fmt.Sprintf("%d", a.cfg.OutputCRF),
		"-preset", a.cfg.Preset,
		"-c:a", "aac",
		"-b:a", a.cfg.AudioBitrate,
		"-shortest",
		"-movflags", "+faststart",
	)

	if srtPath != "" {
		args = append(args, "-vf", fmt.Sprintf("subtitles=%s:force_style='FontSize=20,PrimaryColour=&Hffffff,OutlineColour=&H000000,Outline=2'", escapePath(srtPath)))
	}

	args = append(args, "-y", outputPath)

	return a.run(ctx, args, outputPath)
}

func (a *Assembler) AssembleTextOnVideo(ctx context.Context, bgAsset MediaAsset, text, audioPath, outputDir string) (*AssembleResult, error) {
	ts := fmt.Sprintf("%d", time.Now().UnixNano())
	outputPath := filepath.Join(outputDir, "video_"+ts+".mp4")

	safeText := strings.ReplaceAll(text, "'", "\\'")
	safeText = strings.ReplaceAll(safeText, ":", "\\:")

	args := []string{
		"-i", bgAsset.Path,
		"-i", audioPath,
		"-filter_complex", fmt.Sprintf(
			"[0:v]scale=%d:%d:force_original_aspect_ratio=decrease,pad=%d:%d:(ow-iw)/2:(oh-ih)/2:black,fps=%d,"+
				"drawtext=text='%s':fontsize=56:fontcolor=white:x=(w-text_w)/2:y=h*0.15:"+
				"shadowcolor=black:shadowx=2:shadowy=2:box=1:boxcolor=black@0.4:boxborderw=10[outv]",
			a.cfg.OutputWidth, a.cfg.OutputHeight, a.cfg.OutputWidth, a.cfg.OutputHeight, a.cfg.OutputFPS, safeText,
		),
		"-map", "[outv]",
		"-map", "1:a",
		"-c:v", "libx264",
		"-crf", fmt.Sprintf("%d", a.cfg.OutputCRF),
		"-preset", a.cfg.Preset,
		"-c:a", "aac",
		"-b:a", a.cfg.AudioBitrate,
		"-shortest",
		"-movflags", "+faststart",
		"-y", outputPath,
	}

	return a.run(ctx, args, outputPath)
}

func (a *Assembler) AssembleBRoll(ctx context.Context, clips []MediaAsset, audioPath, srtPath, outputDir string) (*AssembleResult, error) {
	if len(clips) == 0 {
		return nil, fmt.Errorf("no clips provided for b-roll")
	}

	ts := fmt.Sprintf("%d", time.Now().UnixNano())
	outputPath := filepath.Join(outputDir, "video_"+ts+".mp4")

	// Repeat the clip list enough times to cover the narration audio. The
	// concat filter just plays whichever inputs we hand it, so duplicating
	// the slice cycles the b-roll instead of cutting the video short.
	clips = repeatClipsToFitAudio(ctx, a.ffmpegBin, clips, audioPath)

	var args []string
	for _, c := range clips {
		args = append(args, "-i", c.Path)
	}
	args = append(args, "-i", audioPath)

	n := len(clips)
	var filterParts []string
	for i := range clips {
		filterParts = append(filterParts, fmt.Sprintf(
			"[%d:v]scale=%d:%d:force_original_aspect_ratio=decrease,pad=%d:%d:(ow-iw)/2:(oh-ih)/2:black,fps=%d,setsar=1[v%d]",
			i, a.cfg.OutputWidth, a.cfg.OutputHeight, a.cfg.OutputWidth, a.cfg.OutputHeight, a.cfg.OutputFPS, i,
		))
	}
	concatInputs := ""
	for i := range clips {
		concatInputs += fmt.Sprintf("[v%d]", i)
	}
	filterParts = append(filterParts, fmt.Sprintf("%sconcat=n=%d:v=1:a=0[outv]", concatInputs, n))

	args = append(args,
		"-filter_complex", strings.Join(filterParts, ";"),
		"-map", "[outv]",
		"-map", fmt.Sprintf("%d:a", n),
		"-c:v", "libx264",
		"-crf", fmt.Sprintf("%d", a.cfg.OutputCRF),
		"-preset", a.cfg.Preset,
		"-c:a", "aac",
		"-b:a", a.cfg.AudioBitrate,
		"-shortest",
		"-movflags", "+faststart",
		"-y", outputPath,
	)

	return a.run(ctx, args, outputPath)
}

// repeatClipsToFitAudio duplicates the clip slice until total clip runtime
// matches or exceeds the audio length. `-shortest` then trims the excess.
func repeatClipsToFitAudio(ctx context.Context, ffmpegBin string, clips []MediaAsset, audioPath string) []MediaAsset {
	audioDur := probeDuration(ctx, ffmpegBin, audioPath)
	if audioDur <= 0 {
		return clips
	}
	var clipTotal float64
	for _, c := range clips {
		d := c.Duration
		if d <= 0 {
			d = probeDuration(ctx, ffmpegBin, c.Path)
			if d > 0 {
				c.Duration = d
			}
		}
		clipTotal += d
	}
	if clipTotal <= 0 || clipTotal >= audioDur {
		return clips
	}
	repeats := int(audioDur/clipTotal) + 1
	out := make([]MediaAsset, 0, len(clips)*repeats)
	for i := 0; i < repeats; i++ {
		out = append(out, clips...)
	}
	return out
}

func (a *Assembler) run(ctx context.Context, args []string, outputPath string) (*AssembleResult, error) {
	cmd := exec.CommandContext(ctx, a.ffmpegBin, args...)
	var logBuf strings.Builder
	cmd.Stdout = &logBuf
	cmd.Stderr = &logBuf

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg error: %w\nlog: %s", err, logBuf.String())
	}

	info, err := os.Stat(outputPath)
	if err != nil {
		return nil, fmt.Errorf("output file not found: %w", err)
	}

	dur := probeDuration(ctx, a.ffmpegBin, outputPath)

	return &AssembleResult{
		VideoPath:       outputPath,
		DurationSeconds: dur,
		FileSizeBytes:   info.Size(),
		Log:             logBuf.String(),
	}, nil
}

func probeDuration(ctx context.Context, ffmpegBin, path string) float64 {
	ffprobe := strings.Replace(ffmpegBin, "ffmpeg", "ffprobe", 1)
	cmd := exec.CommandContext(ctx, ffprobe,
		"-v", "quiet",
		"-print_format", "compact",
		"-show_entries", "format=duration",
		path,
	)
	out, err := cmd.Output()
	if err != nil {
		return 0
	}
	s := strings.TrimSpace(string(out))
	parts := strings.Split(s, "=")
	if len(parts) == 2 {
		var dur float64
		_, _ = fmt.Sscanf(parts[1], "%f", &dur)
		return dur
	}
	return 0
}

func escapePath(path string) string {
	path = strings.ReplaceAll(path, "\\", "\\\\")
	path = strings.ReplaceAll(path, ":", "\\:")
	path = strings.ReplaceAll(path, "'", "\\'")
	return path
}
