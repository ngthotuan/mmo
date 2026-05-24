"use client";

import { useState, useEffect } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { User, KeyRound, Info, Settings2 } from "lucide-react";
import { toast } from "sonner";
import { Header } from "@/components/layout/Header";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import { Switch } from "@/components/ui/switch";
import { authApi } from "@/lib/api/auth";

export const AUTO_APPROVE_KEY = "pipeline_auto_approve";

export default function SettingsPage() {
  const qc = useQueryClient();

  const { data: user, isLoading } = useQuery({
    queryKey: ["me"],
    queryFn: authApi.me,
  });

  // Profile form
  const [name, setName] = useState("");
  const profileMut = useMutation({
    mutationFn: () => authApi.updateProfile(name),
    onSuccess: () => {
      toast.success("Profile updated");
      qc.invalidateQueries({ queryKey: ["me"] });
    },
    onError: () => toast.error("Failed to update profile"),
  });

  // Pipeline settings
  const [autoApprove, setAutoApprove] = useState(false);
  useEffect(() => {
    setAutoApprove(localStorage.getItem(AUTO_APPROVE_KEY) === "true");
  }, []);
  const toggleAutoApprove = (val: boolean) => {
    setAutoApprove(val);
    localStorage.setItem(AUTO_APPROVE_KEY, String(val));
    toast.success(val ? "Auto-approve enabled" : "Auto-approve disabled");
  };

  // Password form
  const [pw, setPw] = useState({ current: "", next: "", confirm: "" });
  const passwordMut = useMutation({
    mutationFn: () => {
      if (pw.next !== pw.confirm) throw new Error("Passwords do not match");
      return authApi.changePassword(pw.current, pw.next);
    },
    onSuccess: () => {
      toast.success("Password changed");
      setPw({ current: "", next: "", confirm: "" });
    },
    onError: (e: Error) => toast.error(e.message || "Failed to change password"),
  });

  if (isLoading) {
    return (
      <div className="flex flex-col gap-6 p-6">
        <Header title="Settings" description="Manage your account and pipeline preferences" />
        <Skeleton className="h-48 rounded-lg" />
        <Skeleton className="h-56 rounded-lg" />
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-6 p-6 max-w-2xl">
      <Header title="Settings" description="Manage your account and pipeline preferences" />

      {/* Profile */}
      <Card>
        <CardHeader className="flex flex-row items-center gap-2">
          <User className="h-4 w-4 text-muted-foreground" />
          <CardTitle className="text-base">Profile</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-4">
          <div className="flex flex-col gap-1.5">
            <Label className="text-xs">Email</Label>
            <Input value={user?.email ?? ""} disabled className="bg-muted" />
          </div>
          <div className="flex flex-col gap-1.5">
            <Label className="text-xs">Display Name</Label>
            <Input
              placeholder={user?.name}
              value={name}
              onChange={(e) => setName(e.target.value)}
            />
          </div>
          <Button
            className="w-fit"
            disabled={!name.trim() || profileMut.isPending}
            onClick={() => profileMut.mutate()}
          >
            {profileMut.isPending ? "Saving…" : "Save Profile"}
          </Button>
        </CardContent>
      </Card>

      {/* Password */}
      <Card>
        <CardHeader className="flex flex-row items-center gap-2">
          <KeyRound className="h-4 w-4 text-muted-foreground" />
          <CardTitle className="text-base">Change Password</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-4">
          <div className="flex flex-col gap-1.5">
            <Label className="text-xs">Current Password</Label>
            <Input
              type="password"
              value={pw.current}
              onChange={(e) => setPw({ ...pw, current: e.target.value })}
            />
          </div>
          <div className="flex flex-col gap-1.5">
            <Label className="text-xs">New Password</Label>
            <Input
              type="password"
              value={pw.next}
              onChange={(e) => setPw({ ...pw, next: e.target.value })}
            />
          </div>
          <div className="flex flex-col gap-1.5">
            <Label className="text-xs">Confirm New Password</Label>
            <Input
              type="password"
              value={pw.confirm}
              onChange={(e) => setPw({ ...pw, confirm: e.target.value })}
            />
            {pw.next && pw.confirm && pw.next !== pw.confirm && (
              <p className="text-xs text-destructive">Passwords do not match</p>
            )}
          </div>
          <Button
            className="w-fit"
            disabled={
              !pw.current || !pw.next || pw.next !== pw.confirm || passwordMut.isPending
            }
            onClick={() => passwordMut.mutate()}
          >
            {passwordMut.isPending ? "Changing…" : "Change Password"}
          </Button>
        </CardContent>
      </Card>

      {/* Pipeline settings */}
      <Card>
        <CardHeader className="flex flex-row items-center gap-2">
          <Settings2 className="h-4 w-4 text-muted-foreground" />
          <CardTitle className="text-base">Pipeline Settings</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-4">
          <div className="flex items-center justify-between">
            <div className="flex flex-col gap-0.5">
              <Label className="text-sm">Auto-approve scripts</Label>
              <span className="text-xs text-muted-foreground">
                Automatically approve generated scripts and start video creation without manual review
              </span>
            </div>
            <Switch checked={autoApprove} onCheckedChange={toggleAutoApprove} />
          </div>
        </CardContent>
      </Card>

      {/* Account info */}
      <Card>
        <CardHeader className="flex flex-row items-center gap-2">
          <Info className="h-4 w-4 text-muted-foreground" />
          <CardTitle className="text-base">Account Info</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-2 text-sm text-muted-foreground">
          <div className="flex justify-between">
            <span>Role</span>
            <span className="capitalize font-medium text-foreground">{user?.role}</span>
          </div>
          <div className="flex justify-between">
            <span>Member since</span>
            <span className="font-medium text-foreground">
              {user?.created_at
                ? new Date(user.created_at).toLocaleDateString()
                : "—"}
            </span>
          </div>
          <div className="flex justify-between">
            <span>User ID</span>
            <span className="font-mono text-xs">{user?.id?.slice(0, 16)}…</span>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
