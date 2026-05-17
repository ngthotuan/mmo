"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { RefreshCw, TrendingUp, FileText, Loader2 } from "lucide-react";
import { toast } from "sonner";
import { Header } from "@/components/layout/Header";
import { Button } from "@/components/ui/button";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Skeleton } from "@/components/ui/skeleton";
import { Badge } from "@/components/ui/badge";
import { TrendCard } from "@/components/pipeline/TrendCard";
import { ContentPlanCard } from "@/components/pipeline/ContentPlanCard";
import { contentApi } from "@/lib/api/content";

const CONTENT_STATUSES = ["", "draft", "approved", "video_ready", "published", "rejected"];

export default function ContentPage() {
  const qc = useQueryClient();
  const [planStatus, setPlanStatus] = useState("");

  const { data: trendsData, isLoading: trendsLoading } = useQuery({
    queryKey: ["trends", "new"],
    queryFn: () => contentApi.listTrends({ status: "new", page: 1 }),
  });

  const { data: plansData, isLoading: plansLoading } = useQuery({
    queryKey: ["content-plans", planStatus],
    queryFn: () => contentApi.listPlans({ status: planStatus || undefined, page: 1 }),
  });

  const discoverMut = useMutation({
    mutationFn: contentApi.discoverTrends,
    onSuccess: () => {
      toast.success("Discovery queued — trends will appear in a few minutes");
      qc.invalidateQueries({ queryKey: ["trends"] });
    },
    onError: () => toast.error("Failed to queue discovery"),
  });

  return (
    <div className="flex flex-col gap-6 p-6">
      <Header title="Content Pipeline" />

      <Tabs defaultValue="trends">
        <div className="flex items-center justify-between flex-wrap gap-3">
          <TabsList>
            <TabsTrigger value="trends" className="gap-2">
              <TrendingUp className="h-4 w-4" />
              Trends
              {trendsData?.pagination.total ? (
                <Badge variant="secondary" className="ml-1 h-4 px-1 text-xs">
                  {trendsData.pagination.total}
                </Badge>
              ) : null}
            </TabsTrigger>
            <TabsTrigger value="plans" className="gap-2">
              <FileText className="h-4 w-4" />
              Content Plans
              {plansData?.pagination.total ? (
                <Badge variant="secondary" className="ml-1 h-4 px-1 text-xs">
                  {plansData.pagination.total}
                </Badge>
              ) : null}
            </TabsTrigger>
          </TabsList>

          <Button
            variant="outline"
            size="sm"
            className="gap-2"
            onClick={() => discoverMut.mutate()}
            disabled={discoverMut.isPending}
          >
            {discoverMut.isPending
              ? <Loader2 className="h-4 w-4 animate-spin" />
              : <RefreshCw className="h-4 w-4" />
            }
            Discover Trends
          </Button>
        </div>

        {/* ─── Trends Tab ─────────────────────────────────────────────────── */}
        <TabsContent value="trends" className="mt-4">
          {trendsLoading ? (
            <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
              {Array.from({ length: 6 }).map((_, i) => (
                <Skeleton key={i} className="h-36 rounded-lg" />
              ))}
            </div>
          ) : trendsData?.data.length ? (
            <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
              {trendsData.data.map((t) => (
                <TrendCard
                  key={t.id}
                  trend={t}
                  onGenerated={() => qc.invalidateQueries({ queryKey: ["trends"] })}
                />
              ))}
            </div>
          ) : (
            <div className="flex flex-col items-center py-16 text-muted-foreground">
              <TrendingUp className="mb-3 h-10 w-10 opacity-30" />
              <p className="text-sm">No trends discovered yet.</p>
              <Button variant="link" className="mt-1 text-sm" onClick={() => discoverMut.mutate()}>
                Discover now
              </Button>
            </div>
          )}
        </TabsContent>

        {/* ─── Content Plans Tab ──────────────────────────────────────────── */}
        <TabsContent value="plans" className="mt-4">
          {/* Status filter */}
          <div className="mb-4 flex flex-wrap gap-2">
            {CONTENT_STATUSES.map((s) => (
              <Button
                key={s || "all"}
                variant={planStatus === s ? "default" : "outline"}
                size="sm"
                className="h-7 text-xs"
                onClick={() => setPlanStatus(s)}
              >
                {s || "All"}
              </Button>
            ))}
          </div>

          {plansLoading ? (
            <div className="flex flex-col gap-3">
              {Array.from({ length: 4 }).map((_, i) => (
                <Skeleton key={i} className="h-28 rounded-lg" />
              ))}
            </div>
          ) : plansData?.data.length ? (
            <div className="flex flex-col gap-3">
              {plansData.data.map((p) => (
                <ContentPlanCard key={p.id} plan={p} />
              ))}
            </div>
          ) : (
            <div className="flex flex-col items-center py-16 text-muted-foreground">
              <FileText className="mb-3 h-10 w-10 opacity-30" />
              <p className="text-sm">No content plans yet. Generate from a trend.</p>
            </div>
          )}
        </TabsContent>
      </Tabs>
    </div>
  );
}
