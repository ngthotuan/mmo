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

// subtitleForceStyle builds the libass force_style string with a configurable
// font size, positioned in the lower third with a readable outline.
func (a *Assembler) subtitleForceStyle() string {
	size := a.cfg.SubtitleFontSize
	if size <= 0 {
		size = 13
	}
	return fmt.Sprintf("FontSize=%d,PrimaryColour=&Hffffff,OutlineColour=&H000000,BorderStyle=1,Outline=2,Shadow=1,Alignment=2,MarginV=60", size)
}

func (a *Assembler) TempDir(jobID string) (string, error) {
	base := a.cfg.TempDir
	if base == "" {
		base = filepath.Join(os.TempDir(), "mmo-media")
	}
	dir := filepath.Join(base, jobID)
	return dir, os.MkdirAll(dir, 0755)
}

// ProbeDuration returns the duration in seconds of a media file (0 on error).
func (a *Assembler) ProbeDuration(ctx context.Context, path string) float64 {
	return probeDuration(ctx, a.ffmpegBin, path)
}

// ConcatAudio concatenates audio files into a single mp3 (re-encoded so joins
// are gapless). With a single input it is returned unchanged.
func (a *Assembler) ConcatAudio(ctx context.Context, inputs []string, outputDir string) (string, error) {
	if len(inputs) == 0 {
		return "", fmt.Errorf("no audio inputs")
	}
	if len(inputs) == 1 {
		return inputs[0], nil
	}
	out := filepath.Join(outputDir, fmt.Sprintf("narration_%d.mp3", time.Now().UnixNano()))
	var args []string
	for _, in := range inputs {
		args = append(args, "-i", in)
	}
	fc := ""
	for i := range inputs {
		fc += fmt.Sprintf("[%d:a]", i)
	}
	fc += fmt.Sprintf("concat=n=%d:v=0:a=1[a]", len(inputs))
	args = append(args, "-filter_complex", fc, "-map", "[a]", "-c:a", "libmp3lame", "-q:a", "4", "-y", out)

	cmd := exec.CommandContext(ctx, a.ffmpegBin, args...)
	var logBuf strings.Builder
	cmd.Stderr = &logBuf
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("concat audio: %w\n%s", err, logBuf.String())
	}
	return out, nil
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
		args = append(args, "-vf", fmt.Sprintf("subtitles=%s:force_style='%s'", escapePath(srtPath), a.subtitleForceStyle()))
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

// brollSegment is one cut of a source clip: play `Dur` seconds starting at `Start`.
type brollSegment struct {
	Path  string
	Start float64
	Dur   float64
}

func (a *Assembler) AssembleBRoll(ctx context.Context, clips []MediaAsset, audioPath, srtPath, outputDir string) (*AssembleResult, error) {
	if len(clips) == 0 {
		return nil, fmt.Errorf("no clips provided for b-roll")
	}
	audioDur := probeDuration(ctx, a.ffmpegBin, audioPath)
	segs := a.playlistForDuration(ctx, clips, audioDur)
	return a.renderSegments(ctx, segs, audioPath, srtPath, outputDir)
}

// AssembleScenes renders a scene-by-scene video: each scene's own clips fill
// exactly that scene's narration duration, so the visuals track the content.
// Scenes with no footage borrow from a global pool to avoid gaps.
func (a *Assembler) AssembleScenes(ctx context.Context, sceneClips [][]MediaAsset, sceneDur []float64, audioPath, srtPath, outputDir string) (*AssembleResult, error) {
	// Global pool: every clip across scenes, used as fallback for empty scenes.
	var pool []MediaAsset
	for _, cs := range sceneClips {
		pool = append(pool, cs...)
	}
	if len(pool) == 0 {
		return nil, fmt.Errorf("no clips provided for scenes")
	}

	var segs []brollSegment
	for i, clips := range sceneClips {
		var d float64
		if i < len(sceneDur) {
			d = sceneDur[i]
		}
		if d <= 0 {
			continue
		}
		if len(clips) == 0 {
			clips = pool // borrow footage so the scene isn't blank
		}
		segs = append(segs, a.playlistForDuration(ctx, clips, d)...)
	}
	if len(segs) == 0 {
		// No per-scene durations usable → fall back to covering the whole audio.
		segs = a.playlistForDuration(ctx, pool, probeDuration(ctx, a.ffmpegBin, audioPath))
	}
	return a.renderSegments(ctx, segs, audioPath, srtPath, outputDir)
}

// renderSegments builds the ffmpeg graph from a segment playlist: each segment
// is trimmed to its slice, scaled/padded to the output frame, concatenated, and
// the narration audio + burned subtitles are mapped on. `-shortest` trims any
// excess video so the result matches the audio exactly.
func (a *Assembler) renderSegments(ctx context.Context, segs []brollSegment, audioPath, srtPath, outputDir string) (*AssembleResult, error) {
	if len(segs) == 0 {
		return nil, fmt.Errorf("no segments to render")
	}
	ts := fmt.Sprintf("%d", time.Now().UnixNano())
	outputPath := filepath.Join(outputDir, "video_"+ts+".mp4")

	var args []string
	for _, s := range segs {
		args = append(args, "-i", s.Path)
	}
	args = append(args, "-i", audioPath)

	n := len(segs)
	var filterParts []string
	for i, s := range segs {
		filterParts = append(filterParts, fmt.Sprintf(
			"[%d:v]trim=start=%.3f:duration=%.3f,setpts=PTS-STARTPTS,scale=%d:%d:force_original_aspect_ratio=decrease,pad=%d:%d:(ow-iw)/2:(oh-ih)/2:black,fps=%d,setsar=1[v%d]",
			i, s.Start, s.Dur, a.cfg.OutputWidth, a.cfg.OutputHeight, a.cfg.OutputWidth, a.cfg.OutputHeight, a.cfg.OutputFPS, i,
		))
	}
	concatInputs := ""
	for i := range segs {
		concatInputs += fmt.Sprintf("[v%d]", i)
	}
	finalLabel := "outv"
	filterParts = append(filterParts, fmt.Sprintf("%sconcat=n=%d:v=1:a=0[outv]", concatInputs, n))
	if srtPath != "" {
		filterParts = append(filterParts, fmt.Sprintf(
			"[outv]subtitles=%s:force_style='%s'[outsub]",
			escapePath(srtPath), a.subtitleForceStyle()))
		finalLabel = "outsub"
	}

	args = append(args,
		"-filter_complex", strings.Join(filterParts, ";"),
		"-map", "["+finalLabel+"]",
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

// playlistForDuration produces an ordered list of clip slices covering targetDur.
// Variety strategy:
//   - each clip is shown in short slices of `clip_segment_secs`;
//   - reusing a clip advances to its NEXT slice (different footage), wrapping;
//   - each pass over the clip set rotates the start index and reverses on odd
//     passes, so repeats are de-synchronised rather than an identical loop.
func (a *Assembler) playlistForDuration(ctx context.Context, clips []MediaAsset, targetDur float64) []brollSegment {
	if len(clips) == 0 {
		return nil
	}
	seg := float64(a.cfg.ClipSegmentSecs)
	if seg <= 0 {
		seg = 6
	}

	durs := make([]float64, len(clips))
	for i, c := range clips {
		d := c.Duration
		if d <= 0 {
			d = probeDuration(ctx, a.ffmpegBin, c.Path)
		}
		durs[i] = d
	}

	// Unknown target length → one segment per clip (best effort).
	if targetDur <= 0 {
		out := make([]brollSegment, 0, len(clips))
		for i, c := range clips {
			d := durs[i]
			if d <= 0 || d > seg {
				d = seg
			}
			out = append(out, brollSegment{Path: c.Path, Start: 0, Dur: d})
		}
		return out
	}

	const maxSegments = 600
	appear := make([]int, len(clips))
	var out []brollSegment
	total := 0.0
	for pass := 0; total < targetDur && len(out) < maxSegments; pass++ {
		order := make([]int, len(clips))
		for i := range order {
			order[i] = i
		}
		rot := pass % len(clips)
		order = append(order[rot:], order[:rot]...)
		if pass%2 == 1 {
			for l, r := 0, len(order)-1; l < r; l, r = l+1, r-1 {
				order[l], order[r] = order[r], order[l]
			}
		}

		for _, idx := range order {
			d := durs[idx]
			var start, dur float64
			if d <= 0 {
				start, dur = 0, seg
			} else if d <= seg {
				start, dur = 0, d
			} else {
				slots := int(d / seg)
				if slots < 1 {
					slots = 1
				}
				start = float64(appear[idx]%slots) * seg
				dur = seg
			}
			// Don't overshoot the scene by a lot — trim the last slice to fit.
			if remaining := targetDur - total; remaining > 0 && dur > remaining+0.5 {
				dur = remaining
			}
			out = append(out, brollSegment{Path: clips[idx].Path, Start: start, Dur: dur})
			appear[idx]++
			total += dur
			if total >= targetDur || len(out) >= maxSegments {
				break
			}
		}
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
