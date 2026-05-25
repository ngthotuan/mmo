"use client";

import { useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { Loader2, X } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { autoPilotApi, type AutoPilotProfile, type AutoPilotInput } from "@/lib/api/auto-pilot";

interface Props {
  profile: AutoPilotProfile | null;
  onClose: () => void;
  onSaved: () => void;
}

const PLATFORM_OPTIONS = ["tiktok", "facebook"];
const SOURCE_OPTIONS = ["google_trends", "youtube", "reddit", "vnexpress"];
const VOICE_OPTIONS = [
  { value: "vi-VN-HoaiMyNeural", label: "Hoài My (Nữ) - tiếng Việt" },
  { value: "vi-VN-NamMinhNeural", label: "Nam Minh (Nam) - tiếng Việt" },
  { value: "en-US-AriaNeural", label: "Aria (Female) - English" },
  { value: "en-US-GuyNeural", label: "Guy (Male) - English" },
];

export function ProfileFormDialog({ profile, onClose, onSaved }: Props) {
  const isEdit = !!profile;

  const [name, setName] = useState(profile?.name ?? "");
  const [niche, setNiche] = useState(profile?.niche ?? "");
  const [voice, setVoice] = useState(profile?.voice ?? "vi-VN-HoaiMyNeural");
  const [platforms, setPlatforms] = useState<string[]>(profile?.target_platforms ?? ["tiktok"]);
  const [trendFilter, setTrendFilter] = useState(profile?.trend_filter ?? "");
  const [sources, setSources] = useState<string[]>(profile?.trend_sources ?? []);
  const [dailyCount, setDailyCount] = useState(profile?.daily_count ?? 2);
  const [scheduleTimes, setScheduleTimes] = useState<string>(
    (profile?.schedule_times ?? ["09:00", "19:00"]).join(",")
  );
  const [autoApprove, setAutoApprove] = useState(profile?.auto_approve ?? true);
  const [autoPublish, setAutoPublish] = useState(profile?.auto_publish ?? true);
  const [enabled, setEnabled] = useState(profile?.enabled ?? true);

  const saveMut = useMutation({
    mutationFn: async () => {
      const body: AutoPilotInput = {
        name,
        niche,
        voice,
        target_platforms: platforms,
        trend_filter: trendFilter,
        trend_sources: sources,
        daily_count: dailyCount,
        schedule_times: scheduleTimes.split(",").map((s) => s.trim()).filter(Boolean),
        auto_approve: autoApprove,
        auto_publish: autoPublish,
        enabled,
      };
      if (isEdit) return autoPilotApi.update(profile!.id, body);
      return autoPilotApi.create(body);
    },
    onSuccess: () => {
      toast.success(isEdit ? "Đã cập nhật profile" : "Đã tạo profile");
      onSaved();
    },
    onError: (e: unknown) => {
      const msg = e instanceof Error ? e.message : "Lưu thất bại";
      toast.error(msg);
    },
  });

  const togglePlatform = (p: string) =>
    setPlatforms((prev) => (prev.includes(p) ? prev.filter((x) => x !== p) : [...prev, p]));

  const toggleSource = (s: string) =>
    setSources((prev) => (prev.includes(s) ? prev.filter((x) => x !== s) : [...prev, s]));

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
      <div className="w-full max-w-lg max-h-[90vh] overflow-y-auto rounded-lg bg-background shadow-xl">
        <div className="flex items-center justify-between border-b px-6 py-4">
          <h2 className="text-lg font-semibold">
            {isEdit ? "Sửa Auto Pilot Profile" : "Tạo Auto Pilot Profile"}
          </h2>
          <button onClick={onClose} className="text-muted-foreground hover:text-foreground">
            <X className="h-5 w-5" />
          </button>
        </div>

        <div className="flex flex-col gap-4 p-6">
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="name">Tên profile *</Label>
            <Input
              id="name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder='VD: "Review món ăn", "Tin công nghệ"'
            />
          </div>

          <div className="flex flex-col gap-1.5">
            <Label htmlFor="niche">Niche / Chủ đề</Label>
            <Input
              id="niche"
              value={niche}
              onChange={(e) => setNiche(e.target.value)}
              placeholder="VD: food, tech, beauty, finance"
            />
          </div>

          <div className="flex flex-col gap-1.5">
            <Label>Giọng đọc TTS</Label>
            <select
              value={voice}
              onChange={(e) => setVoice(e.target.value)}
              className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
            >
              {VOICE_OPTIONS.map((v) => (
                <option key={v.value} value={v.value}>
                  {v.label}
                </option>
              ))}
            </select>
          </div>

          <div className="flex flex-col gap-1.5">
            <Label>Platforms đăng bài *</Label>
            <div className="flex gap-2">
              {PLATFORM_OPTIONS.map((p) => (
                <button
                  key={p}
                  type="button"
                  onClick={() => togglePlatform(p)}
                  className={`rounded-md border px-3 py-1.5 text-xs font-medium capitalize ${
                    platforms.includes(p)
                      ? "border-violet-500 bg-violet-500/10 text-violet-600"
                      : "border-input bg-background text-muted-foreground"
                  }`}
                >
                  {p}
                </button>
              ))}
            </div>
          </div>

          <div className="flex flex-col gap-1.5">
            <Label>Nguồn trend (để trống = tất cả)</Label>
            <div className="flex flex-wrap gap-2">
              {SOURCE_OPTIONS.map((s) => (
                <button
                  key={s}
                  type="button"
                  onClick={() => toggleSource(s)}
                  className={`rounded-md border px-3 py-1.5 text-xs font-medium ${
                    sources.includes(s)
                      ? "border-violet-500 bg-violet-500/10 text-violet-600"
                      : "border-input bg-background text-muted-foreground"
                  }`}
                >
                  {s}
                </button>
              ))}
            </div>
          </div>

          <div className="flex flex-col gap-1.5">
            <Label htmlFor="trend_filter">Từ khóa lọc trend (substring)</Label>
            <Input
              id="trend_filter"
              value={trendFilter}
              onChange={(e) => setTrendFilter(e.target.value)}
              placeholder='VD: "ăn" để chỉ chọn trend chứa từ "ăn"'
            />
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="daily_count">Video/ngày *</Label>
              <Input
                id="daily_count"
                type="number"
                min={1}
                max={20}
                value={dailyCount}
                onChange={(e) => setDailyCount(parseInt(e.target.value || "1"))}
              />
            </div>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="schedule_times">Lịch chạy (HH:MM, cách nhau ,) *</Label>
              <Input
                id="schedule_times"
                value={scheduleTimes}
                onChange={(e) => setScheduleTimes(e.target.value)}
                placeholder="09:00,19:00"
              />
            </div>
          </div>

          <div className="flex items-center justify-between rounded-lg border px-3 py-2.5">
            <div>
              <p className="text-sm font-medium">Auto duyệt script</p>
              <p className="text-xs text-muted-foreground">Script tự duyệt, không cần xem trước</p>
            </div>
            <Switch checked={autoApprove} onCheckedChange={setAutoApprove} />
          </div>

          <div className="flex items-center justify-between rounded-lg border px-3 py-2.5">
            <div>
              <p className="text-sm font-medium">Auto publish</p>
              <p className="text-xs text-muted-foreground">
                Video tự đăng lên tất cả kênh đã kết nối khi xong
              </p>
            </div>
            <Switch checked={autoPublish} onCheckedChange={setAutoPublish} />
          </div>

          <div className="flex items-center justify-between rounded-lg border px-3 py-2.5">
            <div>
              <p className="text-sm font-medium">Kích hoạt</p>
              <p className="text-xs text-muted-foreground">Bật để cron chạy profile này</p>
            </div>
            <Switch checked={enabled} onCheckedChange={setEnabled} />
          </div>
        </div>

        <div className="flex justify-end gap-2 border-t px-6 py-3">
          <Button variant="ghost" onClick={onClose}>
            Hủy
          </Button>
          <Button onClick={() => saveMut.mutate()} disabled={saveMut.isPending || !name}>
            {saveMut.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : "Lưu"}
          </Button>
        </div>
      </div>
    </div>
  );
}
