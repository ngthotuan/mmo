"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { toast } from "sonner";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { channelsApi } from "@/lib/api/channels";

interface Props {
  open: boolean;
  onClose: () => void;
}

export function ConnectChannelModal({ open, onClose }: Props) {
  const router = useRouter();
  const [loading, setLoading] = useState<"tiktok" | "facebook" | null>(null);

  const connect = async (platform: "tiktok" | "facebook") => {
    setLoading(platform);
    try {
      const authURL = await channelsApi.getAuthURL(platform);
      // Store platform in sessionStorage so the callback page knows what to do
      sessionStorage.setItem("oauth_platform", platform);
      // Redirect to platform OAuth page
      window.location.href = authURL;
    } catch {
      toast.error(`Failed to start ${platform} OAuth`);
      setLoading(null);
    }
  };

  return (
    <Dialog open={open} onOpenChange={(v) => !v && onClose()}>
      <DialogContent className="max-w-sm">
        <DialogHeader>
          <DialogTitle>Connect a Channel</DialogTitle>
          <DialogDescription>
            Choose a platform to connect. You&apos;ll be redirected to authorize access.
          </DialogDescription>
        </DialogHeader>

        <div className="flex flex-col gap-3 pt-2">
          <Button
            className="bg-black text-white hover:bg-black/80 gap-3"
            onClick={() => connect("tiktok")}
            disabled={loading !== null}
          >
            {loading === "tiktok" ? "Redirecting…" : "Connect TikTok"}
          </Button>

          <Button
            className="bg-blue-600 text-white hover:bg-blue-700 gap-3"
            onClick={() => connect("facebook")}
            disabled={loading !== null}
          >
            {loading === "facebook" ? "Redirecting…" : "Connect Facebook Page"}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}
