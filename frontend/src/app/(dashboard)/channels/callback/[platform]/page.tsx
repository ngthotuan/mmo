"use client";

import { Suspense, useEffect, useRef, useState } from "react";
import { useParams, useRouter, useSearchParams } from "next/navigation";
import { toast } from "sonner";
import { Loader2, CheckCircle, XCircle } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { channelsApi } from "@/lib/api/channels";

type Step =
  | "detecting"
  | "tiktok_connecting"
  | "facebook_pages"
  | "facebook_connecting"
  | "done"
  | "error";

function OAuthCallbackContent() {
  const router = useRouter();
  const params = useSearchParams();
  const { platform } = useParams<{ platform: string }>();
  const [step, setStep] = useState<Step>("detecting");
  const [error, setError] = useState("");
  const [pages, setPages] = useState<{ id: string; name: string }[] | null>(null);
  const [fbUserToken, setFbUserToken] = useState("");
  const processed = useRef(false);

  useEffect(() => {
    if (processed.current) return;
    processed.current = true;

    const code = params.get("code");
    const state = params.get("state") ?? "";

    if (!code || !platform) {
      setError("Missing OAuth parameters. Please try again.");
      setStep("error");
      return;
    }

    if (platform === "tiktok") {
      setStep("tiktok_connecting");
      channelsApi
        .connectTikTok(code, state)
        .then(() => {
          setStep("done");
          toast.success("TikTok channel connected!");
          setTimeout(() => router.push("/channels"), 1500);
        })
        .catch((err) => {
          const msg = err?.response?.data?.message ?? "Failed to connect TikTok";
          setError(msg);
          setStep("error");
        });
    } else if (platform === "facebook") {
      setStep("facebook_pages");
      channelsApi
        .getFacebookPages(code)
        .then(({ pages: p, userToken }) => {
          setPages(p);
          setFbUserToken(userToken);
        })
        .catch((err) => {
          const msg = err?.response?.data?.message ?? "Failed to fetch Facebook pages";
          setError(msg);
          setStep("error");
        });
    } else {
      setError(`Unknown platform: ${platform}`);
      setStep("error");
    }
  }, [params, platform, router]);

  const connectFBPage = async (pageId: string) => {
    setStep("facebook_connecting");
    try {
      await channelsApi.connectFacebook(fbUserToken, pageId);
      setStep("done");
      toast.success("Facebook Page connected!");
      setTimeout(() => router.push("/channels"), 1500);
    } catch (err: unknown) {
      const msg =
        (err as { response?: { data?: { message?: string } } })?.response?.data?.message ??
        "Failed to connect Facebook Page";
      setError(msg);
      setStep("error");
    }
  };

  return (
    <div className="flex min-h-screen items-center justify-center p-4">
      <Card className="w-full max-w-md">
        <CardHeader>
          <CardTitle className="text-center">
            {step === "detecting"           && "Processing…"}
            {step === "tiktok_connecting"   && "Connecting TikTok…"}
            {step === "facebook_pages"      && "Select a Facebook Page"}
            {step === "facebook_connecting" && "Connecting Page…"}
            {step === "done"                && "Connected!"}
            {step === "error"               && "Connection Failed"}
          </CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col items-center gap-4">
          {(step === "detecting" || step === "tiktok_connecting" || step === "facebook_connecting") && (
            <Loader2 className="h-8 w-8 animate-spin text-primary" />
          )}

          {step === "done" && (
            <CheckCircle className="h-10 w-10 text-green-500" />
          )}

          {step === "error" && (
            <>
              <XCircle className="h-10 w-10 text-destructive" />
              <p className="text-sm text-center text-muted-foreground">{error}</p>
              <Button onClick={() => router.push("/channels")}>Back to Channels</Button>
            </>
          )}

          {step === "facebook_pages" && (
            <div className="w-full flex flex-col gap-2">
              {pages === null ? (
                <p className="text-sm text-center text-muted-foreground">
                  Loading your Facebook Pages…
                  <Loader2 className="mt-2 h-5 w-5 animate-spin mx-auto" />
                </p>
              ) : pages.length === 0 ? (
                <p className="text-sm text-center text-muted-foreground">
                  No Facebook Pages found. Create a Page on Facebook first, then try again.
                </p>
              ) : (
                <>
                  <p className="text-sm text-muted-foreground text-center mb-2">
                    Choose a page to connect:
                  </p>
                  {pages.map((page) => (
                    <Button
                      key={page.id}
                      variant="outline"
                      className="w-full justify-start"
                      onClick={() => connectFBPage(page.id)}
                    >
                      {page.name}
                    </Button>
                  ))}
                </>
              )}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}

export default function OAuthCallbackPage() {
  return (
    <Suspense fallback={
      <div className="flex min-h-screen items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-primary" />
      </div>
    }>
      <OAuthCallbackContent />
    </Suspense>
  );
}
