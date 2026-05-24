"use client";

import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Wand2, ExternalLink, Loader2 } from "lucide-react";
import { toast } from "sonner";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { contentApi } from "@/lib/api/content";
import type { TrendTopic } from "@/lib/types/api.types";
import { AUTO_APPROVE_KEY } from "@/app/(dashboard)/settings/page";

const sourceColor: Record<string, string> = {
  google_trends: "bg-blue-100 text-blue-700",
  youtube:       "bg-red-100 text-red-700",
  reddit:        "bg-orange-100 text-orange-700",
  tiktok:        "bg-black/10 text-black",
};

interface Props {
  trend: TrendTopic;
  onGenerated?: () => void;
}

export function TrendCard({ trend, onGenerated }: Props) {
  const qc = useQueryClient();

  const genMut = useMutation({
    mutationFn: () =>
      contentApi.createFromTrend({
        topic_id: trend.id,
        platforms: ["tiktok"],
        auto_approve: localStorage.getItem(AUTO_APPROVE_KEY) === "true",
      }),
    onSuccess: () => {
      toast.success("Script generated! Check Content tab.");
      qc.invalidateQueries({ queryKey: ["content-plans"] });
      qc.invalidateQueries({ queryKey: ["trends"] });
      onGenerated?.();
    },
    onError: () => toast.error("Failed to generate script"),
  });

  return (
    <Card className="hover:shadow-sm transition-shadow">
      <CardContent className="p-4 flex flex-col gap-2">
        <div className="flex items-start justify-between gap-2">
          <h3 className="font-medium text-sm leading-snug flex-1">{trend.title}</h3>
          <Badge className={`text-xs shrink-0 ${sourceColor[trend.source ?? ""] ?? "bg-muted text-muted-foreground"}`}>
            {trend.source?.replace(/_/g, " ") ?? "unknown"}
          </Badge>
        </div>

        {trend.description && (
          <p className="text-xs text-muted-foreground line-clamp-2">{trend.description}</p>
        )}

        {trend.keywords?.length > 0 && (
          <div className="flex flex-wrap gap-1">
            {trend.keywords.slice(0, 4).map((kw) => (
              <span key={kw} className="rounded bg-muted px-1.5 py-0.5 text-xs text-muted-foreground">
                {kw}
              </span>
            ))}
          </div>
        )}

        <div className="flex items-center justify-between pt-1">
          <span className="text-xs text-muted-foreground">
            {new Date(trend.discovered_at).toLocaleDateString()}
          </span>
          <div className="flex gap-1">
            {trend.source_url && (
              <Button variant="ghost" size="icon" className="h-7 w-7" asChild>
                <a href={trend.source_url} target="_blank" rel="noopener noreferrer">
                  <ExternalLink className="h-3 w-3" />
                </a>
              </Button>
            )}
            <Button
              size="sm"
              variant="secondary"
              className="h-7 gap-1 text-xs"
              onClick={() => genMut.mutate()}
              disabled={genMut.isPending || trend.status === "used"}
            >
              {genMut.isPending
                ? <Loader2 className="h-3 w-3 animate-spin" />
                : <Wand2 className="h-3 w-3" />
              }
              {trend.status === "used" ? "Used" : "Generate"}
            </Button>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
