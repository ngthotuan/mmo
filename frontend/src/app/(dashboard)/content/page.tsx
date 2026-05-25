"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { RefreshCw, TrendingUp, FileText, Loader2, CheckCircle2, XCircle } from "lucide-react";
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
  const [selectedPlans, setSelectedPlans] = useState<Set<string>>(new Set());
  const [selectedTrends, setSelectedTrends] = useState<Set<string>>(new Set());

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
      toast.success("Đang khám phá xu hướng — sẽ xuất hiện trong vài phút");
      qc.invalidateQueries({ queryKey: ["trends"] });
    },
    onError: () => toast.error("Không thể khởi động khám phá xu hướng"),
  });

  const bulkActionMut = useMutation({
    mutationFn: ({ action, ids }: { action: "approve" | "reject" | "delete"; ids: string[] }) =>
      contentApi.bulkActionPlans(action, ids),
    onSuccess: (res, { action }) => {
      const label = action === "approve" ? "duyệt" : action === "reject" ? "từ chối" : "xóa";
      toast.success(`Đã ${label} ${res.processed} kế hoạch`);
      setSelectedPlans(new Set());
      qc.invalidateQueries({ queryKey: ["content-plans"] });
    },
    onError: () => toast.error("Thao tác hàng loạt thất bại"),
  });

  const bulkRejectTrendsMut = useMutation({
    mutationFn: (ids: string[]) => contentApi.bulkRejectTrends(ids),
    onSuccess: (res) => {
      toast.success(`Đã từ chối ${res.processed} xu hướng`);
      setSelectedTrends(new Set());
      qc.invalidateQueries({ queryKey: ["trends"] });
    },
    onError: () => toast.error("Không thể từ chối xu hướng"),
  });

  const togglePlan = (id: string) =>
    setSelectedPlans((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id); else next.add(id);
      return next;
    });

  const toggleTrend = (id: string) =>
    setSelectedTrends((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id); else next.add(id);
      return next;
    });

  const selectAllTrends = () => {
    const allIds = trendsData?.data.map((t) => t.id) ?? [];
    setSelectedTrends(new Set(allIds));
  };

  return (
    <div className="flex flex-col gap-6 p-6">
      <Header title="Nội dung" description="Khám phá xu hướng Việt Nam và quản lý kịch bản AI của bạn" />

      <Tabs defaultValue="trends">
        <div className="flex items-center justify-between flex-wrap gap-3">
          <TabsList>
            <TabsTrigger value="trends" className="gap-2">
              <TrendingUp className="h-4 w-4" />
              Xu hướng
              {trendsData?.pagination.total ? (
                <Badge variant="secondary" className="ml-1 h-4 px-1 text-xs">
                  {trendsData.pagination.total}
                </Badge>
              ) : null}
            </TabsTrigger>
            <TabsTrigger value="plans" className="gap-2">
              <FileText className="h-4 w-4" />
              Kế hoạch nội dung
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
            Khám phá xu hướng
          </Button>
        </div>

        {/* ─── Trends Tab ─────────────────────────────────────────────────── */}
        <TabsContent value="trends" className="mt-4">
          {/* Bulk action bar for trends */}
          {selectedTrends.size > 0 && (
            <div className="mb-3 flex items-center gap-3 rounded-lg border bg-muted/50 px-4 py-2">
              <span className="text-sm font-medium">{selectedTrends.size} đã chọn</span>
              <Button
                size="sm"
                variant="outline"
                className="h-7 gap-1 text-xs text-destructive hover:text-destructive"
                onClick={() => bulkRejectTrendsMut.mutate([...selectedTrends])}
                disabled={bulkRejectTrendsMut.isPending}
              >
                {bulkRejectTrendsMut.isPending ? <Loader2 className="h-3 w-3 animate-spin" /> : <XCircle className="h-3 w-3" />}
                Từ chối tất cả
              </Button>
              <Button size="sm" variant="ghost" className="h-7 text-xs ml-auto" onClick={() => setSelectedTrends(new Set())}>
                Bỏ chọn
              </Button>
            </div>
          )}

          {trendsLoading ? (
            <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
              {Array.from({ length: 6 }).map((_, i) => (
                <Skeleton key={i} className="h-36 rounded-lg" />
              ))}
            </div>
          ) : trendsData?.data.length ? (
            <>
              <div className="mb-3 flex items-center gap-2">
                <Button variant="ghost" size="sm" className="h-7 text-xs" onClick={selectAllTrends}>
                  Chọn tất cả ({trendsData.data.length})
                </Button>
              </div>
              <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
                {trendsData.data.map((t) => (
                  <div key={t.id} className="relative">
                    <input
                      type="checkbox"
                      checked={selectedTrends.has(t.id)}
                      onChange={() => toggleTrend(t.id)}
                      className="absolute top-3 left-3 z-10 h-4 w-4 cursor-pointer rounded border-gray-300"
                    />
                    <TrendCard
                      trend={t}
                      onGenerated={() => qc.invalidateQueries({ queryKey: ["trends"] })}
                    />
                  </div>
                ))}
              </div>
            </>
          ) : (
            <div className="flex flex-col items-center py-16 text-muted-foreground">
              <TrendingUp className="mb-3 h-10 w-10 opacity-30" />
              <p className="text-sm">Chưa có xu hướng nào được khám phá.</p>
              <Button variant="link" className="mt-1 text-sm" onClick={() => discoverMut.mutate()}>
                Khám phá ngay
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
                onClick={() => { setPlanStatus(s); setSelectedPlans(new Set()); }}
              >
                {s === "" ? "Tất cả" : s === "draft" ? "Bản nháp" : s === "approved" ? "Đã duyệt" : s === "video_ready" ? "Video sẵn sàng" : s === "published" ? "Đã đăng" : "Từ chối"}
              </Button>
            ))}
          </div>

          {/* Bulk action bar */}
          {selectedPlans.size > 0 && (
            <div className="mb-3 flex items-center gap-3 rounded-lg border bg-muted/50 px-4 py-2">
              <span className="text-sm font-medium">{selectedPlans.size} đã chọn</span>
              <Button
                size="sm"
                className="h-7 gap-1 text-xs"
                onClick={() => bulkActionMut.mutate({ action: "approve", ids: [...selectedPlans] })}
                disabled={bulkActionMut.isPending}
              >
                {bulkActionMut.isPending ? <Loader2 className="h-3 w-3 animate-spin" /> : <CheckCircle2 className="h-3 w-3" />}
                Duyệt tất cả
              </Button>
              <Button
                size="sm"
                variant="outline"
                className="h-7 gap-1 text-xs text-destructive hover:text-destructive"
                onClick={() => bulkActionMut.mutate({ action: "reject", ids: [...selectedPlans] })}
                disabled={bulkActionMut.isPending}
              >
                {bulkActionMut.isPending ? <Loader2 className="h-3 w-3 animate-spin" /> : <XCircle className="h-3 w-3" />}
                Từ chối tất cả
              </Button>
              <Button size="sm" variant="ghost" className="h-7 text-xs ml-auto" onClick={() => setSelectedPlans(new Set())}>
                Bỏ chọn
              </Button>
            </div>
          )}

          {plansLoading ? (
            <div className="flex flex-col gap-3">
              {Array.from({ length: 4 }).map((_, i) => (
                <Skeleton key={i} className="h-28 rounded-lg" />
              ))}
            </div>
          ) : plansData?.data.length ? (
            <div className="flex flex-col gap-3">
              {plansData.data.map((p) => (
                <div key={p.id} className="flex items-start gap-2">
                  {p.status === "draft" && (
                    <input
                      type="checkbox"
                      checked={selectedPlans.has(p.id)}
                      onChange={() => togglePlan(p.id)}
                      className="mt-3.5 h-4 w-4 shrink-0 cursor-pointer rounded border-gray-300"
                    />
                  )}
                  <div className="flex-1">
                    <ContentPlanCard plan={p} />
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <div className="flex flex-col items-center py-16 text-muted-foreground">
              <FileText className="mb-3 h-10 w-10 opacity-30" />
              <p className="text-sm">Chưa có kế hoạch nội dung nào. Tạo từ xu hướng.</p>
            </div>
          )}
        </TabsContent>
      </Tabs>
    </div>
  );
}
