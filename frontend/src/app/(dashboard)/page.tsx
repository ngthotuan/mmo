"use client";

import { useQuery } from "@tanstack/react-query";
import { Header } from "@/components/layout/Header";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Badge } from "@/components/ui/badge";
import {
  FileText,
  Video,
  Tv2,
  Send,
  CheckCircle2,
  Loader2,
  AlertCircle,
  Clock,
} from "lucide-react";
import { channelsApi } from "@/lib/api/channels";
import { contentApi } from "@/lib/api/content";
import { videosApi } from "@/lib/api/videos";
import { publishApi } from "@/lib/api/publish";
import { pipelineApi } from "@/lib/api/pipeline";

const JOB_STATUS_ICON: Record<string, React.ReactNode> = {
  done: <CheckCircle2 className="h-3.5 w-3.5 text-green-500" />,
  failed: <AlertCircle className="h-3.5 w-3.5 text-destructive" />,
  assembling: <Loader2 className="h-3.5 w-3.5 animate-spin text-yellow-500" />,
  uploading: <Loader2 className="h-3.5 w-3.5 animate-spin text-blue-500" />,
  tts_generating: <Loader2 className="h-3.5 w-3.5 animate-spin text-purple-500" />,
  media_collecting: <Loader2 className="h-3.5 w-3.5 animate-spin text-orange-500" />,
};

export default function DashboardPage() {
  const { data: channels } = useQuery({
    queryKey: ["channels"],
    queryFn: channelsApi.list,
  });
  const { data: plans } = useQuery({
    queryKey: ["content-plans", ""],
    queryFn: () => contentApi.listPlans(),
  });
  const { data: videos } = useQuery({
    queryKey: ["videos", "done"],
    queryFn: () => videosApi.list({ status: "done" }),
  });
  const { data: published } = useQuery({
    queryKey: ["publish-jobs", "published"],
    queryFn: () => publishApi.list({ status: "published" }),
  });
  const { data: pipeline, isLoading: pipelineLoading } = useQuery({
    queryKey: ["pipeline-status"],
    queryFn: pipelineApi.status,
    refetchInterval: 15_000,
  });

  const summaryStats = [
    { label: "Connected Channels", value: channels?.length ?? "—", icon: Tv2, color: "text-blue-500" },
    { label: "Content Plans", value: plans?.pagination?.total ?? "—", icon: FileText, color: "text-purple-500" },
    { label: "Videos Done", value: videos?.total ?? "—", icon: Video, color: "text-green-500" },
    { label: "Posts Published", value: published?.total ?? "—", icon: Send, color: "text-orange-500" },
  ];

  return (
    <div className="flex flex-col gap-6 p-6">
      <Header title="Dashboard" />

      {/* Stats grid */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {summaryStats.map(({ label, value, icon: Icon, color }) => (
          <Card key={label}>
            <CardHeader className="flex flex-row items-center justify-between pb-2">
              <CardTitle className="text-sm font-medium text-muted-foreground">
                {label}
              </CardTitle>
              <Icon className={`h-4 w-4 ${color}`} />
            </CardHeader>
            <CardContent>
              <p className="text-2xl font-bold">{value}</p>
            </CardContent>
          </Card>
        ))}
      </div>

      {/* Pipeline status */}
      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          <CardTitle className="text-base">Pipeline Status</CardTitle>
          {pipelineLoading && <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />}
        </CardHeader>
        <CardContent>
          {pipelineLoading ? (
            <div className="space-y-2">
              {Array.from({ length: 3 }).map((_, i) => (
                <Skeleton key={i} className="h-8 rounded" />
              ))}
            </div>
          ) : pipeline?.active_jobs?.length === 0 ? (
            <div className="flex flex-col items-center gap-2 py-8 text-muted-foreground">
              <CheckCircle2 className="h-8 w-8 opacity-30" />
              <p className="text-sm">No active jobs. All pipelines are idle.</p>
            </div>
          ) : (
            <div className="space-y-2">
              {/* Status counts */}
              {pipeline?.video_status_counts && (
                <div className="flex flex-wrap gap-2 mb-4">
                  {Object.entries(pipeline.video_status_counts).map(([status, count]) => (
                    <Badge key={status} variant="outline" className="gap-1 text-xs">
                      {JOB_STATUS_ICON[status] || <Clock className="h-3 w-3" />}
                      {status}: {count}
                    </Badge>
                  ))}
                </div>
              )}
              {/* Active jobs list */}
              {pipeline?.active_jobs?.map((job) => (
                <div key={job.id} className="flex items-center gap-3 rounded-md border p-2.5 text-sm">
                  {JOB_STATUS_ICON[job.status] || <Clock className="h-3.5 w-3.5 text-muted-foreground" />}
                  <span className="font-mono text-xs text-muted-foreground">{job.id.slice(0, 8)}…</span>
                  <Badge variant="secondary" className="ml-auto text-xs">{job.status}</Badge>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
