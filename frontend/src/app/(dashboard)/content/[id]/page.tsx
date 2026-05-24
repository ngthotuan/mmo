"use client";

import { useState } from "react";
import { use } from "react";
import { useRouter } from "next/navigation";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { ArrowLeft, Save, Wand2, CheckCircle, XCircle, Loader2 } from "lucide-react";
import { toast } from "sonner";
import { Header } from "@/components/layout/Header";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { contentApi } from "@/lib/api/content";

interface Props {
  params: Promise<{ id: string }>;
}

export default function ContentDetailPage({ params }: Props) {
  const { id } = use(params);
  const router = useRouter();
  const qc = useQueryClient();

  const { data: plan, isLoading } = useQuery({
    queryKey: ["content-plan", id],
    queryFn: () => contentApi.getPlan(id),
  });

  const [script, setScript] = useState("");
  const [title, setTitle] = useState("");
  const [voice, setVoice] = useState("");
  const dirty = !!plan && (script !== plan.script || title !== plan.title || voice !== (plan.voice ?? ""));

  // Sync local state when plan loads
  if (plan && script === "" && title === "") {
    setScript(plan.script);
    setTitle(plan.title);
    setVoice(plan.voice ?? "");
  }

  const saveMut = useMutation({
    mutationFn: () => contentApi.updatePlan(id, { title, script, voice: voice || undefined }),
    onSuccess: () => {
      toast.success("Saved");
      qc.invalidateQueries({ queryKey: ["content-plan", id] });
    },
    onError: () => toast.error("Failed to save"),
  });

  const approveMut = useMutation({
    mutationFn: () => contentApi.approvePlan(id),
    onSuccess: () => {
      toast.success("Approved — video creation queued!");
      qc.invalidateQueries({ queryKey: ["content-plan", id] });
      router.push("/content");
    },
    onError: () => toast.error("Failed to approve"),
  });

  const rejectMut = useMutation({
    mutationFn: () => contentApi.rejectPlan(id),
    onSuccess: () => {
      toast.success("Plan rejected");
      router.push("/content");
    },
    onError: () => toast.error("Failed to reject"),
  });

  const regenMut = useMutation({
    mutationFn: () => contentApi.regenerateScript(id),
    onSuccess: (updated) => {
      toast.success("Script regenerated");
      setScript(updated.script);
      setTitle(updated.title);
      qc.invalidateQueries({ queryKey: ["content-plan", id] });
    },
    onError: () => toast.error("Failed to regenerate"),
  });

  if (isLoading) {
    return (
      <div className="p-6 flex flex-col gap-4">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-10 w-full" />
        <Skeleton className="h-64 w-full" />
      </div>
    );
  }

  if (!plan) return null;

  const isDraft = plan.status === "draft";
  const busy = saveMut.isPending || approveMut.isPending || rejectMut.isPending || regenMut.isPending;

  return (
    <div className="flex flex-col gap-6 p-6">
      <Header title="Edit Content Plan" />

      <div className="flex items-center gap-3">
        <Button variant="ghost" size="sm" onClick={() => router.back()} className="gap-2">
          <ArrowLeft className="h-4 w-4" /> Back
        </Button>
        <Badge>{plan.status.replace("_", " ")}</Badge>
        <span className="text-xs text-muted-foreground">
          {plan.niche || "General"} · {plan.target_platforms?.join(", ")}
        </span>
      </div>

      <div className="grid gap-4 lg:grid-cols-3">
        {/* ─── Script Editor ──────────────────────────────────────────── */}
        <div className="lg:col-span-2 flex flex-col gap-4">
          <Card>
            <CardHeader className="pb-3">
              <CardTitle className="text-base">Script</CardTitle>
            </CardHeader>
            <CardContent className="flex flex-col gap-3">
              <div className="flex flex-col gap-1.5">
                <Label htmlFor="title">Title</Label>
                <Input
                  id="title"
                  value={title}
                  onChange={(e) => setTitle(e.target.value)}
                  disabled={!isDraft}
                />
              </div>
              <div className="flex flex-col gap-1.5">
                <Label htmlFor="script">Script</Label>
                <textarea
                  id="script"
                  className="min-h-[240px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50 resize-y"
                  value={script}
                  onChange={(e) => setScript(e.target.value)}
                  disabled={!isDraft}
                  placeholder="Video script will appear here..."
                />
              </div>
              {isDraft && (
                <div className="flex gap-2">
                  <Button size="sm" onClick={() => saveMut.mutate()} disabled={!dirty || busy} className="gap-2">
                    {saveMut.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : <Save className="h-4 w-4" />}
                    Save
                  </Button>
                  <Button size="sm" variant="outline" onClick={() => regenMut.mutate()} disabled={busy} className="gap-2">
                    {regenMut.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : <Wand2 className="h-4 w-4" />}
                    Regenerate
                  </Button>
                </div>
              )}
            </CardContent>
          </Card>
        </div>

        {/* ─── Metadata + Actions ─────────────────────────────────────── */}
        <div className="flex flex-col gap-4">
          <Card>
            <CardHeader className="pb-3">
              <CardTitle className="text-base">Script Metadata</CardTitle>
            </CardHeader>
            <CardContent className="flex flex-col gap-3 text-sm">
              {plan.script_metadata?.hook && (
                <div>
                  <p className="font-medium text-xs text-muted-foreground mb-1">Hook</p>
                  <p className="text-sm">{plan.script_metadata.hook}</p>
                </div>
              )}
              {plan.script_metadata?.cta && (
                <div>
                  <p className="font-medium text-xs text-muted-foreground mb-1">CTA</p>
                  <p className="text-sm">{plan.script_metadata.cta}</p>
                </div>
              )}
              {plan.script_metadata?.hashtags?.length > 0 && (
                <div>
                  <p className="font-medium text-xs text-muted-foreground mb-1">Hashtags</p>
                  <div className="flex flex-wrap gap-1">
                    {plan.script_metadata.hashtags.map((tag) => (
                      <span key={tag} className="rounded bg-muted px-1.5 py-0.5 text-xs">
                        #{tag}
                      </span>
                    ))}
                  </div>
                </div>
              )}
              {plan.script_metadata?.caption && (
                <div>
                  <p className="font-medium text-xs text-muted-foreground mb-1">Caption</p>
                  <p className="text-sm text-muted-foreground">{plan.script_metadata.caption}</p>
                </div>
              )}
            </CardContent>
          </Card>

          {isDraft && (
            <>
              <Card>
                <CardHeader className="pb-3">
                  <CardTitle className="text-base">Voice</CardTitle>
                </CardHeader>
                <CardContent>
                  <select
                    value={voice}
                    onChange={(e) => setVoice(e.target.value)}
                    className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus:outline-none focus:ring-2 focus:ring-ring"
                  >
                    <option value="">Default (en-US-AriaNeural)</option>
                    <optgroup label="English (US)">
                      <option value="en-US-AriaNeural">Aria (Female)</option>
                      <option value="en-US-JennyNeural">Jenny (Female)</option>
                      <option value="en-US-GuyNeural">Guy (Male)</option>
                      <option value="en-US-DavisNeural">Davis (Male)</option>
                      <option value="en-US-AmberNeural">Amber (Female)</option>
                      <option value="en-US-ChristopherNeural">Christopher (Male)</option>
                    </optgroup>
                    <optgroup label="English (UK)">
                      <option value="en-GB-SoniaNeural">Sonia (Female)</option>
                      <option value="en-GB-RyanNeural">Ryan (Male)</option>
                    </optgroup>
                    <optgroup label="English (AU)">
                      <option value="en-AU-NatashaNeural">Natasha (Female)</option>
                      <option value="en-AU-WilliamNeural">William (Male)</option>
                    </optgroup>
                    <optgroup label="Vietnamese">
                      <option value="vi-VN-HoaiMyNeural">Hoài My (Female)</option>
                      <option value="vi-VN-NamMinhNeural">Nam Minh (Male)</option>
                    </optgroup>
                  </select>
                  <p className="mt-1 text-xs text-muted-foreground">Used for TTS narration</p>
                </CardContent>
              </Card>

              <Card>
                <CardHeader className="pb-3">
                  <CardTitle className="text-base">Actions</CardTitle>
                </CardHeader>
                <CardContent className="flex flex-col gap-2">
                  <Button className="w-full gap-2" onClick={() => approveMut.mutate()} disabled={busy}>
                    {approveMut.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : <CheckCircle className="h-4 w-4" />}
                    Approve & Create Video
                  </Button>
                  <Button variant="outline" className="w-full gap-2 text-destructive hover:text-destructive"
                    onClick={() => rejectMut.mutate()} disabled={busy}>
                    {rejectMut.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : <XCircle className="h-4 w-4" />}
                    Reject
                  </Button>
                </CardContent>
              </Card>
            </>
          )}
        </div>
      </div>
    </div>
  );
}
