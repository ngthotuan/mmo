"use client";

import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Trash2, ToggleLeft, ToggleRight, ExternalLink } from "lucide-react";
import { toast } from "sonner";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { channelsApi } from "@/lib/api/channels";
import type { Channel } from "@/lib/types/api.types";

interface Props {
  channel: Channel;
}

const platformLabel: Record<string, string> = {
  tiktok:   "TikTok",
  facebook: "Facebook",
  youtube:  "YouTube",
};

const platformColor: Record<string, string> = {
  tiktok:   "bg-black text-white",
  facebook: "bg-blue-600 text-white",
  youtube:  "bg-red-600 text-white",
};

export function ChannelCard({ channel: ch }: Props) {
  const qc = useQueryClient();
  const [toggling, setToggling] = useState(false);

  const deleteMut = useMutation({
    mutationFn: () => channelsApi.delete(ch.id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["channels"] });
      toast.success("Channel disconnected");
    },
    onError: () => toast.error("Failed to disconnect channel"),
  });

  const toggleMut = useMutation({
    mutationFn: (active: boolean) => channelsApi.toggle(ch.id, active),
    onMutate: () => setToggling(true),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["channels"] });
    },
    onSettled: () => setToggling(false),
    onError: () => toast.error("Failed to update channel"),
  });

  return (
    <Card>
      <CardContent className="flex items-center gap-4 p-4">
        <Avatar className="h-12 w-12">
          <AvatarImage src={ch.avatar_url} alt={ch.display_name} />
          <AvatarFallback className="text-sm font-bold">
            {ch.display_name?.charAt(0).toUpperCase() ?? "?"}
          </AvatarFallback>
        </Avatar>

        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 flex-wrap">
            <span className="font-medium truncate">{ch.display_name}</span>
            <Badge className={`text-xs ${platformColor[ch.platform] ?? ""}`}>
              {platformLabel[ch.platform] ?? ch.platform}
            </Badge>
            {!ch.is_active && (
              <Badge variant="outline" className="text-xs text-muted-foreground">
                Inactive
              </Badge>
            )}
            {ch.dry_run && (
              <Badge variant="outline" className="text-xs text-amber-600 border-amber-400">
                Dry-run
              </Badge>
            )}
          </div>
          <p className="text-sm text-muted-foreground truncate">@{ch.username || ch.platform_user_id}</p>
        </div>

        <div className="flex items-center gap-1 shrink-0">
          <Button
            variant="ghost"
            size="icon"
            disabled={toggling}
            onClick={() => toggleMut.mutate(!ch.is_active)}
            title={ch.is_active ? "Deactivate" : "Activate"}
          >
            {ch.is_active
              ? <ToggleRight className="h-5 w-5 text-green-500" />
              : <ToggleLeft  className="h-5 w-5 text-muted-foreground" />
            }
          </Button>

          {ch.platform === "tiktok" && (
            <Button variant="ghost" size="icon" asChild>
              <a
                href={`https://www.tiktok.com/@${ch.username}`}
                target="_blank"
                rel="noopener noreferrer"
                title="View on TikTok"
              >
                <ExternalLink className="h-4 w-4" />
              </a>
            </Button>
          )}

          <Button
            variant="ghost"
            size="icon"
            className="text-destructive hover:text-destructive"
            onClick={() => {
              if (confirm(`Disconnect ${ch.display_name}?`)) {
                deleteMut.mutate();
              }
            }}
            disabled={deleteMut.isPending}
            title="Disconnect"
          >
            <Trash2 className="h-4 w-4" />
          </Button>
        </div>
      </CardContent>
    </Card>
  );
}
