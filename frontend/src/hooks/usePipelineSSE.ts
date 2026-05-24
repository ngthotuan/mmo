"use client";

import { useEffect, useRef } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { getAccessToken } from "@/lib/api/client";

const API_URL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

interface JobEvent {
  id: string;
  content_plan_id: string;
  status: string;
  output_video_url?: string;
}

/**
 * Subscribes to the backend SSE pipeline events stream.
 * On each "jobs" event, invalidates the relevant React Query caches so
 * video and content plan lists refresh without explicit polling.
 */
export function usePipelineSSE() {
  const qc = useQueryClient();
  const esRef = useRef<EventSource | null>(null);

  useEffect(() => {
    const token = getAccessToken();
    if (!token || typeof window === "undefined") return;

    const url = `${API_URL}/api/v1/pipeline/events?token=${encodeURIComponent(token)}`;
    const es = new EventSource(url);
    esRef.current = es;

    es.addEventListener("jobs", (e: MessageEvent) => {
      try {
        const jobs: JobEvent[] = JSON.parse(e.data);
        const hasActiveJob = jobs.some(
          (j) => j.status !== "done" && j.status !== "failed"
        );

        // Always refresh video list to get latest statuses
        qc.invalidateQueries({ queryKey: ["videos"] });

        // If any job just completed, refresh content plans and publish data too
        if (!hasActiveJob && jobs.length > 0) {
          qc.invalidateQueries({ queryKey: ["content-plans"] });
          qc.invalidateQueries({ queryKey: ["publish-jobs"] });
        }
      } catch {
        // ignore parse errors
      }
    });

    es.onerror = () => {
      // EventSource reconnects automatically; no action needed
    };

    return () => {
      es.close();
      esRef.current = null;
    };
  }, [qc]);
}
