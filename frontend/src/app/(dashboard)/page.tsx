"use client";

import { useQuery } from "@tanstack/react-query";
import { Header } from "@/components/layout/Header";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";
import {
  FileText,
  Video,
  Tv2,
  Send,
  CheckCircle2,
  Loader2,
  AlertCircle,
  Clock,
  Activity,
} from "lucide-react";
import { channelsApi } from "@/lib/api/channels";
import { contentApi } from "@/lib/api/content";
import { videosApi } from "@/lib/api/videos";
import { publishApi } from "@/lib/api/publish";
import { pipelineApi } from "@/lib/api/pipeline";

const JOB_STATUS_ICON: Record<string, React.ReactNode> = {
  done: <CheckCircle2 className="h-3.5 w-3.5 text-green-500" />,
  failed: <AlertCircle className="h-3.5 w-3.5 text-red-500" />,
  assembling: <Loader2 className="h-3.5 w-3.5 animate-spin text-yellow-500" />,
  uploading: <Loader2 className="h-3.5 w-3.5 animate-spin text-blue-500" />,
  tts_generating: <Loader2 className="h-3.5 w-3.5 animate-spin text-purple-500" />,
  media_collecting: <Loader2 className="h-3.5 w-3.5 animate-spin text-orange-500" />,
};

const STATUS_COLOR: Record<string, string> = {
  done: "bg-green-50 text-green-700 ring-green-200",
  failed: "bg-red-50 text-red-700 ring-red-200",
  assembling: "bg-yellow-50 text-yellow-700 ring-yellow-200",
  uploading: "bg-blue-50 text-blue-700 ring-blue-200",
  tts_generating: "bg-purple-50 text-purple-700 ring-purple-200",
  media_collecting: "bg-orange-50 text-orange-700 ring-orange-200",
};

const summaryConfig = [
  {
    label: "Connected Channels",
    icon: Tv2,
    gradient: "from-blue-500 to-cyan-500",
    iconBg: "bg-blue-50",
    iconColor: "text-blue-600",
    key: "channels" as const,
  },
  {
    label: "Content Plans",
    icon: FileText,
    gradient: "from-violet-500 to-purple-500",
    iconBg: "bg-violet-50",
    iconColor: "text-violet-600",
    key: "plans" as const,
  },
  {
    label: "Videos Done",
    icon: Video,
    gradient: "from-emerald-500 to-green-500",
    iconBg: "bg-emerald-50",
    iconColor: "text-emerald-600",
    key: "videos" as const,
  },
  {
    label: "Posts Published",
    icon: Send,
    gradient: "from-orange-500 to-amber-500",
    iconBg: "bg-orange-50",
    iconColor: "text-orange-600",
    key: "published" as const,
  },
];

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

  const statValues: Record<string, number | string> = {
    channels: channels?.length ?? "—",
    plans: plans?.pagination?.total ?? "—",
    videos: videos?.total ?? "—",
    published: published?.total ?? "—",
  };

  return (
    <div className="flex flex-col gap-6 p-6">
      <Header title="Dashboard" description="Your content automation overview" />

      {/* Stats grid */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {summaryConfig.map(({ label, icon: Icon, iconBg, iconColor, key }) => (
          <Card key={label} className="border-0 shadow-sm hover:shadow-md transition-shadow duration-200">
            <CardContent className="p-5">
              <div className="flex items-start justify-between">
                <div>
                  <p className="text-sm font-medium text-muted-foreground">{label}</p>
                  <p className="mt-2 text-3xl font-bold text-slate-900">
                    {statValues[key]}
                  </p>
                </div>
                <div className={cn("rounded-xl p-2.5", iconBg)}>
                  <Icon className={cn("h-5 w-5", iconColor)} />
                </div>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>

      {/* Pipeline status */}
      <Card className="border-0 shadow-sm">
        <CardHeader className="flex flex-row items-center gap-2 pb-3">
          <Activity className="h-4 w-4 text-violet-500" />
          <CardTitle className="text-base font-semibold">Pipeline Status</CardTitle>
          {pipelineLoading && (
            <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
          )}
        </CardHeader>
        <CardContent>
          {pipelineLoading ? (
            <div className="space-y-2">
              {Array.from({ length: 3 }).map((_, i) => (
                <Skeleton key={i} className="h-10 rounded-lg" />
              ))}
            </div>
          ) : pipeline?.active_jobs?.length === 0 ? (
            <div className="flex flex-col items-center gap-2 rounded-xl bg-slate-50 py-10 text-muted-foreground">
              <div className="flex h-12 w-12 items-center justify-center rounded-full bg-slate-100">
                <CheckCircle2 className="h-6 w-6 text-slate-400" />
              </div>
              <p className="text-sm font-medium">All pipelines idle</p>
              <p className="text-xs text-slate-400">No active jobs running</p>
            </div>
          ) : (
            <div className="space-y-3">
              {pipeline?.video_status_counts && (
                <div className="flex flex-wrap gap-2">
                  {Object.entries(pipeline.video_status_counts).map(
                    ([status, count]) => (
                      <span
                        key={status}
                        className={cn(
                          "inline-flex items-center gap-1.5 rounded-full px-2.5 py-1 text-xs font-medium ring-1",
                          STATUS_COLOR[status] ?? "bg-slate-50 text-slate-700 ring-slate-200"
                        )}
                      >
                        {JOB_STATUS_ICON[status] || (
                          <Clock className="h-3 w-3" />
                        )}
                        {status}: {count}
                      </span>
                    )
                  )}
                </div>
              )}
              <div className="flex flex-col gap-2">
                {pipeline?.active_jobs?.map((job) => (
                  <div
                    key={job.id}
                    className="flex items-center gap-3 rounded-lg border border-slate-100 bg-white px-4 py-3 text-sm shadow-sm"
                  >
                    {JOB_STATUS_ICON[job.status] || (
                      <Clock className="h-3.5 w-3.5 text-muted-foreground" />
                    )}
                    <span className="font-mono text-xs text-muted-foreground">
                      {job.id.slice(0, 8)}…
                    </span>
                    <Badge
                      variant="secondary"
                      className="ml-auto text-xs capitalize"
                    >
                      {job.status.replace("_", " ")}
                    </Badge>
                  </div>
                ))}
              </div>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
