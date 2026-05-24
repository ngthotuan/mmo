"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  Send,
  CheckCircle2,
  XCircle,
  Clock,
  Loader2,
  AlertCircle,
  ShoppingBag,
} from "lucide-react";
import { toast } from "sonner";
import { Header } from "@/components/layout/Header";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { publishApi } from "@/lib/api/publish";
import { productsApi } from "@/lib/api/products";
import type { PublishJob, PublishJobStatus, Product } from "@/lib/types/api.types";

const STATUS_LABELS: Record<PublishJobStatus, string> = {
  scheduled: "Scheduled",
  publishing: "Publishing",
  published: "Published",
  failed: "Failed",
  cancelled: "Cancelled",
};

const STATUS_VARIANT: Record<
  PublishJobStatus,
  "default" | "secondary" | "destructive" | "outline"
> = {
  scheduled: "outline",
  publishing: "secondary",
  published: "default",
  failed: "destructive",
  cancelled: "outline",
};

const PUBLISH_STATUSES: Array<{ value: string; label: string }> = [
  { value: "", label: "All" },
  { value: "scheduled", label: "Scheduled" },
  { value: "published", label: "Published" },
  { value: "failed", label: "Failed" },
];

function PublishJobCard({
  job,
  onPublishNow,
  onCancel,
  onTagProducts,
}: {
  job: PublishJob;
  onPublishNow: () => void;
  onCancel: () => void;
  onTagProducts: () => void;
}) {
  const scheduledAt = job.scheduled_at
    ? new Date(job.scheduled_at).toLocaleString()
    : "—";
  const publishedAt = job.published_at
    ? new Date(job.published_at).toLocaleString()
    : null;

  return (
    <div className="rounded-lg border bg-card p-4 flex flex-col gap-3">
      <div className="flex items-start justify-between gap-2">
        <div className="flex flex-col min-w-0">
          <span className="text-sm font-medium capitalize">{job.platform}</span>
          <span className="text-xs text-muted-foreground truncate">
            {job.caption?.slice(0, 60) || "No caption"}
          </span>
        </div>
        <Badge variant={STATUS_VARIANT[job.status]}>
          {STATUS_LABELS[job.status]}
        </Badge>
      </div>

      <div className="text-xs text-muted-foreground space-y-0.5">
        <div className="flex items-center gap-1">
          <Clock className="h-3 w-3" />
          Scheduled: {scheduledAt}
        </div>
        {publishedAt && (
          <div className="flex items-center gap-1">
            <CheckCircle2 className="h-3 w-3" />
            Published: {publishedAt}
          </div>
        )}
        {job.hashtags?.length > 0 && (
          <div className="text-primary truncate">
            {job.hashtags.slice(0, 4).map((t) => `#${t}`).join(" ")}
            {job.hashtags.length > 4 ? " …" : ""}
          </div>
        )}
      </div>

      {job.status === "failed" && job.error_message && (
        <div className="flex items-start gap-1.5 rounded-md bg-destructive/10 p-2 text-xs text-destructive">
          <AlertCircle className="h-3.5 w-3.5 mt-0.5 shrink-0" />
          <span className="line-clamp-2">{job.error_message}</span>
        </div>
      )}

      {job.status === "publishing" && (
        <div className="flex items-center gap-2 text-xs text-muted-foreground">
          <Loader2 className="h-3 w-3 animate-spin" />
          Publishing…
        </div>
      )}

      {job.platform_post_url && (
        <a
          href={job.platform_post_url}
          target="_blank"
          rel="noopener noreferrer"
          className="text-xs text-primary hover:underline"
        >
          View post →
        </a>
      )}

      <div className="flex gap-2 flex-wrap">
        {(job.status === "scheduled" || job.status === "failed") && (
          <Button size="sm" variant="outline" onClick={onPublishNow}>
            <Send className="h-3.5 w-3.5 mr-1" />
            Publish Now
          </Button>
        )}
        {job.status === "scheduled" && (
          <Button size="sm" variant="outline" onClick={onTagProducts}>
            <ShoppingBag className="h-3.5 w-3.5 mr-1" />
            Products
          </Button>
        )}
        {job.status === "scheduled" && (
          <Button
            size="sm"
            variant="ghost"
            className="text-destructive hover:text-destructive ml-auto"
            onClick={onCancel}
          >
            <XCircle className="h-3.5 w-3.5 mr-1" />
            Cancel
          </Button>
        )}
      </div>
    </div>
  );
}

function TagProductsDialog({
  jobId,
  open,
  onClose,
}: {
  jobId: string;
  open: boolean;
  onClose: () => void;
}) {
  const qc = useQueryClient();
  const { data: allProducts } = useQuery({
    queryKey: ["products"],
    queryFn: () => productsApi.list(),
    enabled: open,
  });
  const { data: tagged } = useQuery({
    queryKey: ["job-products", jobId],
    queryFn: () => productsApi.listByPublishJob(jobId),
    enabled: open && !!jobId,
  });

  const taggedIds = new Set((tagged ?? []).map((p: Product) => p.id));
  const [selected, setSelected] = useState<Set<string>>(() => new Set());

  const toggle = (id: string) =>
    setSelected((prev) => {
      const next = new Set(prev);
      next.has(id) ? next.delete(id) : next.add(id);
      return next;
    });

  const { mutate, isPending } = useMutation({
    mutationFn: () =>
      productsApi.tagPublishJob(jobId, [
        ...Array.from(taggedIds),
        ...Array.from(selected),
      ].filter((id, i, arr) => arr.indexOf(id) === i)),
    onSuccess: () => {
      toast.success("Products tagged");
      qc.invalidateQueries({ queryKey: ["job-products", jobId] });
      onClose();
    },
    onError: () => toast.error("Failed to tag products"),
  });

  const products = allProducts?.data ?? [];

  return (
    <Dialog open={open} onOpenChange={(o) => !o && onClose()}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>Tag Products</DialogTitle>
        </DialogHeader>
        <div className="flex flex-col gap-2 max-h-72 overflow-y-auto">
          {products.length === 0 ? (
            <p className="text-sm text-muted-foreground text-center py-6">
              No products in catalog. Add products first.
            </p>
          ) : (
            products.map((p: Product) => {
              const isTagged = taggedIds.has(p.id);
              const isSelected = selected.has(p.id);
              return (
                <label
                  key={p.id}
                  className="flex items-center gap-3 rounded-md border p-2.5 cursor-pointer hover:bg-accent"
                >
                  <input
                    type="checkbox"
                    className="h-4 w-4"
                    defaultChecked={isTagged}
                    checked={isTagged || isSelected}
                    onChange={() => !isTagged && toggle(p.id)}
                    disabled={isTagged}
                  />
                  {p.cover_image_url ? (
                    <img src={p.cover_image_url} alt="" className="h-8 w-8 rounded object-cover" />
                  ) : (
                    <ShoppingBag className="h-8 w-8 text-muted-foreground" />
                  )}
                  <div className="flex-1 min-w-0">
                    <p className="text-sm font-medium line-clamp-1">{p.name}</p>
                    <p className="text-xs text-muted-foreground capitalize">{p.platform}</p>
                  </div>
                  {isTagged && <Badge variant="secondary" className="text-xs">Tagged</Badge>}
                </label>
              );
            })
          )}
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={onClose}>Close</Button>
          {selected.size > 0 && (
            <Button onClick={() => mutate()} disabled={isPending}>
              {isPending ? "Saving…" : `Tag ${selected.size} product(s)`}
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

export default function PublishPage() {
  const qc = useQueryClient();
  const [statusFilter, setStatusFilter] = useState("");
  const [tagJobId, setTagJobId] = useState<string | null>(null);

  const { data, isLoading } = useQuery({
    queryKey: ["publish-jobs", statusFilter],
    queryFn: () => publishApi.list({ status: statusFilter || undefined }),
    refetchInterval: 60_000, // SSE in layout handles live updates
  });

  const publishNowMut = useMutation({
    mutationFn: publishApi.publishNow,
    onSuccess: () => {
      toast.success("Queued for immediate publishing");
      qc.invalidateQueries({ queryKey: ["publish-jobs"] });
    },
    onError: () => toast.error("Failed to publish"),
  });

  const cancelMut = useMutation({
    mutationFn: publishApi.cancel,
    onSuccess: () => {
      toast.success("Cancelled");
      qc.invalidateQueries({ queryKey: ["publish-jobs"] });
    },
    onError: () => toast.error("Failed to cancel"),
  });

  const jobs = data?.data ?? [];
  const total = data?.total ?? 0;

  return (
    <div className="flex flex-col gap-6 p-6">
      <Header title="Publish Queue" description="Schedule and monitor your content publishing jobs" />

      <div className="flex items-center justify-between flex-wrap gap-3">
        <div className="flex gap-2 flex-wrap">
          {PUBLISH_STATUSES.map(({ value, label }) => (
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
            <Skeleton key={i} className="h-48 rounded-lg" />
          ))}
        </div>
      ) : jobs.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-24 text-muted-foreground gap-3">
          <Send className="h-12 w-12 opacity-30" />
          <p className="text-sm">No publish jobs yet. Schedule a video from the Videos page.</p>
        </div>
      ) : (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {jobs.map((job) => (
            <PublishJobCard
              key={job.id}
              job={job}
              onPublishNow={() => publishNowMut.mutate(job.id)}
              onCancel={() => cancelMut.mutate(job.id)}
              onTagProducts={() => setTagJobId(job.id)}
            />
          ))}
        </div>
      )}

      {tagJobId && (
        <TagProductsDialog
          jobId={tagJobId}
          open={!!tagJobId}
          onClose={() => setTagJobId(null)}
        />
      )}
    </div>
  );
}
