"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { PlusCircle, Tv2 } from "lucide-react";
import { Header } from "@/components/layout/Header";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { ChannelCard } from "@/components/channel/ChannelCard";
import { ConnectChannelModal } from "@/components/channel/ConnectChannelModal";
import { channelsApi } from "@/lib/api/channels";

export default function ChannelsPage() {
  const [showModal, setShowModal] = useState(false);

  const { data: channels, isLoading } = useQuery({
    queryKey: ["channels"],
    queryFn: channelsApi.list,
  });

  return (
    <div className="flex flex-col gap-6 p-6">
      <Header title="Channels" />

      <div className="flex items-center justify-between">
        <p className="text-sm text-muted-foreground">
          {channels?.length ?? 0} channel{channels?.length !== 1 ? "s" : ""} connected
        </p>
        <Button onClick={() => setShowModal(true)} className="gap-2">
          <PlusCircle className="h-4 w-4" />
          Connect Channel
        </Button>
      </div>

      {isLoading ? (
        <div className="flex flex-col gap-3">
          {[1, 2, 3].map((i) => (
            <Skeleton key={i} className="h-20 w-full rounded-lg" />
          ))}
        </div>
      ) : channels && channels.length > 0 ? (
        <div className="flex flex-col gap-3">
          {channels.map((ch) => (
            <ChannelCard key={ch.id} channel={ch} />
          ))}
        </div>
      ) : (
        <div className="flex flex-col items-center justify-center rounded-xl border border-dashed py-16 text-muted-foreground">
          <Tv2 className="mb-3 h-10 w-10 opacity-30" />
          <p className="text-sm">No channels connected yet.</p>
          <Button
            variant="link"
            className="mt-1 text-sm"
            onClick={() => setShowModal(true)}
          >
            Connect your first channel
          </Button>
        </div>
      )}

      <ConnectChannelModal open={showModal} onClose={() => setShowModal(false)} />
    </div>
  );
}
