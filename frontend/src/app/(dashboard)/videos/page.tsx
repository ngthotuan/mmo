"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  Video,
  RefreshCw,
  Download,
  Trash2,
  AlertCircle,
  Loader2,
  Play,
} from "lucide-react";
import { toast } from "sonner";
import { Header } from "@/components/layout/Header";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
import { videosApi } from "@/lib/api/videos";
import type { VideoJob, VideoJobStatus } from "@/lib/types/api.types";

const STATUS_LABELS: Record<VideoJobStatus, string> = {
  pending: "Pending",
  media_collecting: "Collecting Media",
  tts_generating: "Generating TTS",
  assembling: "Assembling",
  uploading: "Uploading",
  done: "Done",
  failed: "Failed",
};

const STATUS_VARIANT: Record<
  VideoJobStatus,
  "default" | "secondary" | "destructive" | "outline"
> = {
  pending: "outline",
  media_collecting: "secondary",
  tts_generating: "secondary",
  assembling: "secondary",
  uploading: "secondary",
  done: "default",
  failed: "destructive",
};

const VIDEO_STATUSES: Array<{ value: string; label: string }> = [
  { value: "", label: "All" },
  { value: "done", label: "Done" },
  { value: "assembling", label: "Assembling" },
  { value: "failed", label: "Failed" },
];

function formatBytes(bytes: number): string {
  if (!bytes) return "—";
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(0)} KB`;
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
}

function formatDuration(secs: number): string {
  if (!secs) return "—";
  const m = Math.floor(secs / 60);
  const s = Math.floor(secs % 60);
  return `${m}:${s.toString().padStart(2, "0")}`;
}

function VideoCard({
  job,
  onPreview,
  onRetry,
  onDeleteRequest,
  onDownload,
}: {
  job: VideoJob;
  onPreview: (job: VideoJob) => void;
  onRetry: (id: string) => void;
  onDeleteRequest: (id: string) => void;
  onDownload: (id: string) => void;
}) {
  return (
    <div className="rounded-lg border bg-card p-4 flex flex-col gap-3">
      <div className="flex items-start justify-between gap-2">
        <div className="flex items-center gap-2 min-w-0">
          <Video className="h-4 w-4 shrink-0 text-muted-foreground" />
          <span className="text-sm font-medium truncate">{job.id.slice(0, 8)}…</span>
        </div>
        <Badge variant={STATUS_VARIANT[job.status]}>
          {STATUS_LABELS[job.status]}
        </Badge>
      </div>

      <div className="grid grid-cols-2 gap-1 text-xs text-muted-foreground">
        <span>Duration: {formatDuration(job.duration_seconds)}</span>
        <span>Size: {formatBytes(job.file_size_bytes)}</span>
        {job.retry_count > 0 && (
          <span className="col-span-2">Retries: {job.retry_count}</span>
        )}
      </div>

      {job.status === "failed" && job.error_message && (
        <div className="flex items-start gap-1.5 rounded-md bg-destructive/10 p-2 text-xs text-destructive">
          <AlertCircle className="h-3.5 w-3.5 mt-0.5 shrink-0" />
          <span className="line-clamp-2">{job.error_message}</span>
        </div>
      )}

      {(["media_collecting", "tts_generating", "assembling", "uploading"] as VideoJobStatus[]).includes(
        job.status
      ) && (
        <div className="flex items-center gap-2 text-xs text-muted-foreground">
          <Loader2 className="h-3 w-3 animate-spin" />
          Processing…
        </div>
      )}

      <div className="flex gap-2 flex-wrap">
        {job.status === "done" && (
          <>
            <Button size="sm" variant="outline" onClick={() => onPreview(job)}>
              <Play className="h-3.5 w-3.5 mr-1" />
              Preview
            </Button>
            <Button size="sm" variant="outline" onClick={() => onDownload(job.id)}>
              <Download className="h-3.5 w-3.5 mr-1" />
              Download
            </Button>
          </>
        )}
        {job.status === "failed" && (
          <Button size="sm" variant="outline" onClick={() => onRetry(job.id)}>
            <RefreshCw className="h-3.5 w-3.5 mr-1" />
            Retry
          </Button>
        )}
        <Button
          size="sm"
          variant="ghost"
          className="text-destructive hover:text-destructive ml-auto"
          onClick={() => onDeleteRequest(job.id)}
        >
          <Trash2 className="h-3.5 w-3.5" />
        </Button>
      </div>
    </div>
  );
}

export default function VideosPage() {
  const qc = useQueryClient();
  const [statusFilter, setStatusFilter] = useState("");
  const [previewJob, setPreviewJob] = useState<VideoJob | null>(null);
  const [previewURL, setPreviewURL] = useState<string>("");
  const [deleteTarget, setDeleteTarget] = useState<string | null>(null);

  const { data, isLoading } = useQuery({
    queryKey: ["videos", statusFilter],
    queryFn: () => videosApi.list({ status: statusFilter || undefined }),
    refetchInterval: 10_000,
  });

  const retryMut = useMutation({
    mutationFn: videosApi.retry,
    onSuccess: () => {
      toast.success("Retry queued");
      qc.invalidateQueries({ queryKey: ["videos"] });
    },
    onError: () => toast.error("Failed to retry"),
  });

  const deleteMut = useMutation({
    mutationFn: videosApi.delete,
    onSuccess: () => {
      toast.success("Video deleted");
      setDeleteTarget(null);
      qc.invalidateQueries({ queryKey: ["videos"] });
    },
    onError: () => toast.error("Failed to delete"),
  });

  const handlePreview = async (job: VideoJob) => {
    if (job.output_video_url) {
      setPreviewURL(job.output_video_url);
      setPreviewJob(job);
      return;
    }
    try {
      const url = await videosApi.getDownloadURL(job.id);
      setPreviewURL(url);
      setPreviewJob(job);
    } catch {
      toast.error("Failed to get video URL");
    }
  };

  const handleDownload = async (id: string) => {
    try {
      const url = await videosApi.getDownloadURL(id);
      const a = document.createElement("a");
      a.href = url;
      a.download = `video-${id}.mp4`;
      a.click();
    } catch {
      toast.error("Failed to get download URL");
    }
  };

  const jobs = data?.data ?? [];
  const total = data?.total ?? 0;

  return (
    <div className="flex flex-col gap-6 p-6">
      <Header title="Videos" />

      <div className="flex items-center justify-between flex-wrap gap-3">
        <div className="flex gap-2 flex-wrap">
          {VIDEO_STATUSES.map(({ value, label }) => (
            <Button
              key={value}
              size="sm"
              variant={statusFilter === value ? "default" : "outline"}
              onClick={() => setStatusFilter(value)}
            >
              {label}
            </Button>
          ))}
        </div>
        <span className="text-sm text-muted-foreground">{total} jobs</span>
      </div>

      {isLoading ? (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {Array.from({ length: 6 }).map((_, i) => (
            <Skeleton key={i} className="h-40 rounded-lg" />
          ))}
        </div>
      ) : jobs.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-24 text-muted-foreground gap-3">
          <Video className="h-12 w-12 opacity-30" />
          <p className="text-sm">No video jobs yet. Approve a content plan to start the pipeline.</p>
        </div>
      ) : (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {jobs.map((job) => (
            <VideoCard
              key={job.id}
              job={job}
              onPreview={handlePreview}
              onRetry={(id) => retryMut.mutate(id)}
              onDeleteRequest={(id) => setDeleteTarget(id)}
              onDownload={handleDownload}
            />
          ))}
        </div>
      )}

      {/* Video preview dialog */}
      <Dialog open={!!previewJob} onOpenChange={(open) => !open && setPreviewJob(null)}>
        <DialogContent className="max-w-sm p-4">
          <DialogHeader>
            <DialogTitle>Video Preview</DialogTitle>
          </DialogHeader>
          {previewURL && (
            <video
              src={previewURL}
              controls
              autoPlay
              className="w-full rounded-md"
              style={{ maxHeight: "70vh" }}
            />
          )}
        </DialogContent>
      </Dialog>

      {/* Delete confirmation dialog */}
      <Dialog open={!!deleteTarget} onOpenChange={(open) => !open && setDeleteTarget(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete video job?</DialogTitle>
          </DialogHeader>
          <p className="text-sm text-muted-foreground">
            This will permanently remove the video and any uploaded files. This cannot be undone.
          </p>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteTarget(null)}>
              Cancel
            </Button>
            <Button
              variant="destructive"
              disabled={deleteMut.isPending}
              onClick={() => deleteTarget && deleteMut.mutate(deleteTarget)}
            >
              {deleteMut.isPending ? <Loader2 className="h-4 w-4 animate-spin mr-2" /> : null}
              Delete
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
