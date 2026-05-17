"use client";

import { useState, useMemo } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  ChevronLeft,
  ChevronRight,
  CalendarDays,
  Loader2,
  CheckCircle2,
  XCircle,
  Clock,
  Send,
} from "lucide-react";
import { toast } from "sonner";
import { Header } from "@/components/layout/Header";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { publishApi } from "@/lib/api/publish";
import type { PublishJob, PublishJobStatus } from "@/lib/types/api.types";

const STATUS_ICON: Record<PublishJobStatus, React.ReactNode> = {
  scheduled: <Clock className="h-3 w-3" />,
  publishing: <Loader2 className="h-3 w-3 animate-spin" />,
  published: <CheckCircle2 className="h-3 w-3" />,
  failed: <XCircle className="h-3 w-3" />,
  cancelled: <XCircle className="h-3 w-3" />,
};

const STATUS_COLOR: Record<PublishJobStatus, string> = {
  scheduled: "bg-blue-500/20 text-blue-700 dark:text-blue-300",
  publishing: "bg-yellow-500/20 text-yellow-700 dark:text-yellow-300",
  published: "bg-green-500/20 text-green-700 dark:text-green-300",
  failed: "bg-red-500/20 text-red-700 dark:text-red-300",
  cancelled: "bg-gray-500/20 text-gray-500",
};

const DAYS = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"];
const MONTHS = [
  "January", "February", "March", "April", "May", "June",
  "July", "August", "September", "October", "November", "December",
];

function getCalendarDays(year: number, month: number): Date[] {
  const first = new Date(year, month, 1);
  const last = new Date(year, month + 1, 0);
  const days: Date[] = [];

  // Padding days from previous month
  for (let i = 0; i < first.getDay(); i++) {
    days.push(new Date(year, month, -first.getDay() + i + 1));
  }
  for (let d = 1; d <= last.getDate(); d++) {
    days.push(new Date(year, month, d));
  }
  // Pad to full weeks
  while (days.length % 7 !== 0) {
    days.push(new Date(year, month + 1, days.length - last.getDate() - first.getDay() + 1));
  }
  return days;
}

function isSameDay(a: Date, b: Date): boolean {
  return a.getFullYear() === b.getFullYear() &&
    a.getMonth() === b.getMonth() &&
    a.getDate() === b.getDate();
}

export default function SchedulePage() {
  const qc = useQueryClient();
  const today = new Date();
  const [current, setCurrent] = useState({ year: today.getFullYear(), month: today.getMonth() });

  const rangeStart = new Date(current.year, current.month, 1).toISOString();
  const rangeEnd = new Date(current.year, current.month + 1, 0, 23, 59, 59).toISOString();

  const { data: jobs = [], isLoading } = useQuery({
    queryKey: ["calendar", current.year, current.month],
    queryFn: () => publishApi.calendar(rangeStart, rangeEnd),
  });

  const publishNowMut = useMutation({
    mutationFn: publishApi.publishNow,
    onSuccess: () => {
      toast.success("Queued for immediate publishing");
      qc.invalidateQueries({ queryKey: ["calendar"] });
    },
    onError: () => toast.error("Failed to publish"),
  });

  const cancelMut = useMutation({
    mutationFn: publishApi.cancel,
    onSuccess: () => {
      toast.success("Cancelled");
      qc.invalidateQueries({ queryKey: ["calendar"] });
    },
    onError: () => toast.error("Failed to cancel"),
  });

  const calDays = useMemo(
    () => getCalendarDays(current.year, current.month),
    [current.year, current.month]
  );

  function jobsOnDay(day: Date): PublishJob[] {
    return jobs.filter((j) => j.scheduled_at && isSameDay(new Date(j.scheduled_at), day));
  }

  function prevMonth() {
    setCurrent((c) => {
      if (c.month === 0) return { year: c.year - 1, month: 11 };
      return { year: c.year, month: c.month - 1 };
    });
  }

  function nextMonth() {
    setCurrent((c) => {
      if (c.month === 11) return { year: c.year + 1, month: 0 };
      return { year: c.year, month: c.month + 1 };
    });
  }

  return (
    <div className="flex flex-col gap-6 p-6">
      <Header title="Content Calendar" />

      <div className="rounded-lg border bg-card">
        {/* Calendar header */}
        <div className="flex items-center justify-between p-4 border-b">
          <div className="flex items-center gap-2">
            <CalendarDays className="h-5 w-5 text-muted-foreground" />
            <h2 className="font-semibold">
              {MONTHS[current.month]} {current.year}
            </h2>
            {isLoading && <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />}
          </div>
          <div className="flex gap-1">
            <Button size="sm" variant="outline" onClick={prevMonth}>
              <ChevronLeft className="h-4 w-4" />
            </Button>
            <Button size="sm" variant="outline" onClick={() => setCurrent({ year: today.getFullYear(), month: today.getMonth() })}>
              Today
            </Button>
            <Button size="sm" variant="outline" onClick={nextMonth}>
              <ChevronRight className="h-4 w-4" />
            </Button>
          </div>
        </div>

        {/* Day headers */}
        <div className="grid grid-cols-7 border-b">
          {DAYS.map((d) => (
            <div key={d} className="py-2 text-center text-xs font-medium text-muted-foreground">
              {d}
            </div>
          ))}
        </div>

        {/* Calendar grid */}
        <div className="grid grid-cols-7">
          {calDays.map((day, idx) => {
            const isCurrentMonth = day.getMonth() === current.month;
            const isToday = isSameDay(day, today);
            const dayJobs = jobsOnDay(day);

            return (
              <div
                key={idx}
                className={[
                  "min-h-[80px] p-1 border-b border-r",
                  isCurrentMonth ? "" : "opacity-30",
                  idx % 7 === 0 ? "border-l" : "",
                ].join(" ")}
              >
                <div className={[
                  "text-xs font-medium mb-1 w-6 h-6 flex items-center justify-center rounded-full",
                  isToday ? "bg-primary text-primary-foreground" : "text-muted-foreground",
                ].join(" ")}>
                  {day.getDate()}
                </div>
                <div className="flex flex-col gap-0.5">
                  {dayJobs.map((job) => (
                    <CalendarEvent
                      key={job.id}
                      job={job}
                      onPublishNow={() => publishNowMut.mutate(job.id)}
                      onCancel={() => cancelMut.mutate(job.id)}
                    />
                  ))}
                </div>
              </div>
            );
          })}
        </div>
      </div>

      {/* Legend */}
      <div className="flex flex-wrap gap-3 text-xs text-muted-foreground">
        {(Object.keys(STATUS_COLOR) as PublishJobStatus[]).map((s) => (
          <div key={s} className="flex items-center gap-1">
            <span className={`rounded px-1.5 py-0.5 ${STATUS_COLOR[s]}`}>{s}</span>
          </div>
        ))}
      </div>
    </div>
  );
}

function CalendarEvent({
  job,
  onPublishNow,
  onCancel,
}: {
  job: PublishJob;
  onPublishNow: () => void;
  onCancel: () => void;
}) {
  const [expanded, setExpanded] = useState(false);
  const time = job.scheduled_at
    ? new Date(job.scheduled_at).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })
    : "";

  return (
    <div
      className={`rounded px-1 py-0.5 cursor-pointer text-xs ${STATUS_COLOR[job.status]}`}
      onClick={() => setExpanded((v) => !v)}
    >
      <div className="flex items-center gap-1 truncate">
        {STATUS_ICON[job.status]}
        <span className="truncate">{job.platform} {time}</span>
      </div>
      {expanded && (
        <div className="mt-1 flex gap-1" onClick={(e) => e.stopPropagation()}>
          {job.status === "scheduled" && (
            <>
              <button
                className="rounded bg-primary px-1.5 py-0.5 text-primary-foreground hover:opacity-80"
                onClick={onPublishNow}
              >
                <Send className="h-2.5 w-2.5" />
              </button>
              <button
                className="rounded bg-destructive px-1.5 py-0.5 text-destructive-foreground hover:opacity-80"
                onClick={onCancel}
              >
                <XCircle className="h-2.5 w-2.5" />
              </button>
            </>
          )}
        </div>
      )}
    </div>
  );
}

// Suppress unused import warning
const _Badge = Badge;
