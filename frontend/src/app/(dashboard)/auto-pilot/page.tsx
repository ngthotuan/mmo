"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Plus, Play, Trash2, Loader2, Bot, Pencil } from "lucide-react";
import { toast } from "sonner";
import { Header } from "@/components/layout/Header";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { Switch } from "@/components/ui/switch";
import { autoPilotApi, type AutoPilotProfile } from "@/lib/api/auto-pilot";
import { ProfileFormDialog } from "./ProfileFormDialog";

export default function AutoPilotPage() {
  const qc = useQueryClient();
  const [editing, setEditing] = useState<AutoPilotProfile | null>(null);
  const [creating, setCreating] = useState(false);

  const { data, isLoading } = useQuery({
    queryKey: ["auto-pilot"],
    queryFn: autoPilotApi.list,
  });

  const toggleMut = useMutation({
    mutationFn: ({ id, enabled }: { id: string; enabled: boolean }) =>
      autoPilotApi.toggle(id, enabled),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["auto-pilot"] });
    },
  });

  const runMut = useMutation({
    mutationFn: (id: string) => autoPilotApi.runNow(id),
    onSuccess: (res) => {
      toast.success(`Đã tạo ${res.plans_created} kế hoạch`);
      qc.invalidateQueries({ queryKey: ["auto-pilot"] });
      qc.invalidateQueries({ queryKey: ["content-plans"] });
    },
    onError: () => toast.error("Chạy thủ công thất bại"),
  });

  const deleteMut = useMutation({
    mutationFn: (id: string) => autoPilotApi.delete(id),
    onSuccess: () => {
      toast.success("Đã xóa profile");
      qc.invalidateQueries({ queryKey: ["auto-pilot"] });
    },
  });

  return (
    <div className="flex flex-col gap-6 p-6">
      <Header
        title="Auto Pilot"
        description="Cấu hình kênh ảo để app tự động sản xuất video theo lịch — không cần thao tác thủ công"
      />

      <div className="flex justify-end">
        <Button className="gap-2" onClick={() => setCreating(true)}>
          <Plus className="h-4 w-4" />
          Tạo Profile mới
        </Button>
      </div>

      {isLoading ? (
        <div className="grid gap-3 md:grid-cols-2">
          {Array.from({ length: 4 }).map((_, i) => (
            <Skeleton key={i} className="h-48 rounded-lg" />
          ))}
        </div>
      ) : data?.data.length ? (
        <div className="grid gap-3 md:grid-cols-2">
          {data.data.map((p) => (
            <Card key={p.id} className={p.enabled ? "" : "opacity-60"}>
              <CardHeader className="pb-3">
                <div className="flex items-start justify-between gap-2">
                  <div className="min-w-0 flex-1">
                    <CardTitle className="text-base flex items-center gap-2">
                      <Bot className="h-4 w-4 text-violet-500" />
                      <span className="truncate">{p.name}</span>
                    </CardTitle>
                    <p className="text-xs text-muted-foreground mt-1">
                      {p.niche || "general"} · {p.target_platforms.join(", ")}
                    </p>
                  </div>
                  <Switch
                    checked={p.enabled}
                    onCheckedChange={(checked) => toggleMut.mutate({ id: p.id, enabled: checked })}
                  />
                </div>
              </CardHeader>
              <CardContent className="flex flex-col gap-3 text-sm">
                <div className="grid grid-cols-2 gap-2 text-xs">
                  <div>
                    <p className="text-muted-foreground">Lịch chạy</p>
                    <p className="font-medium">{p.schedule_times.join(", ") || "—"}</p>
                  </div>
                  <div>
                    <p className="text-muted-foreground">Video/ngày</p>
                    <p className="font-medium">{p.daily_count}</p>
                  </div>
                  <div>
                    <p className="text-muted-foreground">Tổng đã tạo</p>
                    <p className="font-medium">{p.total_videos}</p>
                  </div>
                  <div>
                    <p className="text-muted-foreground">Lần chạy gần nhất</p>
                    <p className="font-medium">
                      {p.last_run_at
                        ? new Date(p.last_run_at).toLocaleString("vi-VN")
                        : "Chưa chạy"}
                    </p>
                  </div>
                </div>

                <div className="flex flex-wrap gap-1">
                  {p.trend_filter && (
                    <Badge variant="secondary" className="text-xs">
                      Lọc: {p.trend_filter}
                    </Badge>
                  )}
                  {p.auto_approve && <Badge className="text-xs">Auto duyệt</Badge>}
                  {p.auto_publish && <Badge className="text-xs">Auto publish</Badge>}
                </div>

                <div className="flex gap-2 pt-2">
                  <Button
                    size="sm"
                    variant="outline"
                    className="gap-1.5"
                    onClick={() => runMut.mutate(p.id)}
                    disabled={runMut.isPending}
                  >
                    {runMut.isPending ? (
                      <Loader2 className="h-3 w-3 animate-spin" />
                    ) : (
                      <Play className="h-3 w-3" />
                    )}
                    Chạy ngay
                  </Button>
                  <Button
                    size="sm"
                    variant="ghost"
                    className="gap-1.5"
                    onClick={() => setEditing(p)}
                  >
                    <Pencil className="h-3 w-3" />
                    Sửa
                  </Button>
                  <Button
                    size="sm"
                    variant="ghost"
                    className="ml-auto text-destructive hover:text-destructive"
                    onClick={() => {
                      if (confirm(`Xóa profile "${p.name}"?`)) deleteMut.mutate(p.id);
                    }}
                  >
                    <Trash2 className="h-3 w-3" />
                  </Button>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      ) : (
        <div className="flex flex-col items-center py-16 text-muted-foreground">
          <Bot className="mb-3 h-12 w-12 opacity-30" />
          <p className="text-sm">Chưa có Auto Pilot profile nào.</p>
          <p className="text-xs mt-1">
            Tạo profile đầu tiên để app tự động sản xuất video theo lịch.
          </p>
          <Button variant="link" className="mt-2" onClick={() => setCreating(true)}>
            Tạo ngay
          </Button>
        </div>
      )}

      {(creating || editing) && (
        <ProfileFormDialog
          profile={editing}
          onClose={() => {
            setCreating(false);
            setEditing(null);
          }}
          onSaved={() => {
            setCreating(false);
            setEditing(null);
            qc.invalidateQueries({ queryKey: ["auto-pilot"] });
          }}
        />
      )}
    </div>
  );
}
