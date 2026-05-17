"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useRouter } from "next/navigation";
import { CheckCircle, XCircle, RefreshCw, Trash2, ChevronRight } from "lucide-react";
import { toast } from "sonner";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { contentApi } from "@/lib/api/content";
import type { ContentPlan, ContentStatus } from "@/lib/types/api.types";

const statusColor: Record<ContentStatus, string> = {
  draft:        "bg-yellow-100 text-yellow-700",
  approved:     "bg-blue-100 text-blue-700",
  rejected:     "bg-red-100 text-red-700",
  video_queued: "bg-purple-100 text-purple-700",
  video_ready:  "bg-indigo-100 text-indigo-700",
  scheduled:    "bg-cyan-100 text-cyan-700",
  published:    "bg-green-100 text-green-700",
};

interface Props {
  plan: ContentPlan;
}

export function ContentPlanCard({ plan }: Props) {
  const router = useRouter();
  const qc = useQueryClient();

  const approveMut = useMutation({
    mutationFn: () => contentApi.approvePlan(plan.id),
    onSuccess: () => { toast.success("Plan approved — video creation queued"); qc.invalidateQueries({ queryKey: ["content-plans"] }); },
    onError: () => toast.error("Failed to approve"),
  });

  const rejectMut = useMutation({
    mutationFn: () => contentApi.rejectPlan(plan.id),
    onSuccess: () => { toast.success("Plan rejected"); qc.invalidateQueries({ queryKey: ["content-plans"] }); },
    onError: () => toast.error("Failed to reject"),
  });

  const regenMut = useMutation({
    mutationFn: () => contentApi.regenerateScript(plan.id),
    onSuccess: () => { toast.success("Script regenerated"); qc.invalidateQueries({ queryKey: ["content-plans"] }); },
    onError: () => toast.error("Failed to regenerate"),
  });

  const deleteMut = useMutation({
    mutationFn: () => contentApi.deletePlan(plan.id),
    onSuccess: () => { toast.success("Plan deleted"); qc.invalidateQueries({ queryKey: ["content-plans"] }); },
    onError: () => toast.error("Failed to delete"),
  });

  const isDraft = plan.status === "draft";
  const busy = approveMut.isPending || rejectMut.isPending || regenMut.isPending || deleteMut.isPending;

  return (
    <Card className="hover:shadow-sm transition-shadow cursor-pointer"
      onClick={() => router.push(`/content/${plan.id}`)}>
      <CardContent className="p-4 flex flex-col gap-2">
        <div className="flex items-start justify-between gap-2">
          <h3 className="font-medium text-sm leading-snug flex-1 line-clamp-2">{plan.title}</h3>
          <Badge className={`text-xs shrink-0 ${statusColor[plan.status]}`}>
            {plan.status.replace("_", " ")}
          </Badge>
        </div>

        {plan.script && (
          <p className="text-xs text-muted-foreground line-clamp-3 bg-muted/50 rounded p-2">
            {plan.script}
          </p>
        )}

        <div className="flex items-center justify-between pt-1" onClick={e => e.stopPropagation()}>
          <span className="text-xs text-muted-foreground">
            {plan.niche || "General"} · {plan.target_platforms?.join(", ")}
          </span>
          <div className="flex gap-1">
            {isDraft && (
              <>
                <Button size="icon" variant="ghost" className="h-7 w-7 text-green-600 hover:text-green-700"
                  onClick={() => approveMut.mutate()} disabled={busy} title="Approve">
                  <CheckCircle className="h-4 w-4" />
                </Button>
                <Button size="icon" variant="ghost" className="h-7 w-7 text-destructive hover:text-destructive"
                  onClick={() => rejectMut.mutate()} disabled={busy} title="Reject">
                  <XCircle className="h-4 w-4" />
                </Button>
                <Button size="icon" variant="ghost" className="h-7 w-7"
                  onClick={() => regenMut.mutate()} disabled={busy} title="Regenerate script">
                  <RefreshCw className="h-4 w-4" />
                </Button>
              </>
            )}
            {(isDraft || plan.status === "rejected") && (
              <Button size="icon" variant="ghost" className="h-7 w-7 text-destructive hover:text-destructive"
                onClick={() => confirm("Delete this plan?") && deleteMut.mutate()} disabled={busy} title="Delete">
                <Trash2 className="h-4 w-4" />
              </Button>
            )}
            <Button size="icon" variant="ghost" className="h-7 w-7">
              <ChevronRight className="h-4 w-4" />
            </Button>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
