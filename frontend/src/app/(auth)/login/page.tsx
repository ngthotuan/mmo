"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Zap, Loader2, Eye, EyeOff } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { authApi } from "@/lib/api/auth";

export default function LoginPage() {
  const router = useRouter();
  const [mode, setMode] = useState<"login" | "register">("login");
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [showPassword, setShowPassword] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError("");
    try {
      if (mode === "login") {
        await authApi.login(email, password);
      } else {
        await authApi.register(name, email, password);
      }
      router.push("/");
    } catch (err: unknown) {
      const msg =
        (err as { response?: { data?: { message?: string } } })?.response?.data
          ?.message ?? "Something went wrong";
      setError(msg);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="flex min-h-screen items-center justify-center bg-gradient-to-br from-slate-900 via-slate-800 to-indigo-950 p-4">
      <div className="w-full max-w-md">
        {/* Brand */}
        <div className="mb-8 flex flex-col items-center gap-3">
          <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-gradient-to-br from-violet-500 to-indigo-600 shadow-2xl shadow-violet-500/30">
            <Zap className="h-6 w-6 text-white" />
          </div>
          <div className="text-center">
            <h1 className="text-2xl font-bold text-white">AutoContent</h1>
            <p className="text-sm text-slate-400">Social Media Automation Platform</p>
          </div>
        </div>

        {/* Card */}
        <div className="rounded-2xl border border-slate-700/50 bg-slate-800/60 p-8 shadow-2xl backdrop-blur-sm">
          <div className="mb-6 text-center">
            <h2 className="text-lg font-semibold text-white">
              {mode === "login" ? "Welcome back" : "Create your account"}
            </h2>
            <p className="mt-1 text-sm text-slate-400">
              {mode === "login"
                ? "Sign in to manage your content pipeline"
                : "Get started automating your content"}
            </p>
          </div>

          <form onSubmit={handleSubmit} className="flex flex-col gap-4">
            {mode === "register" && (
              <div className="flex flex-col gap-1.5">
                <Label htmlFor="name" className="text-sm font-medium text-slate-300">
                  Name
                </Label>
                <Input
                  id="name"
                  placeholder="Your name"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  required
                  minLength={2}
                  className="border-slate-700 bg-slate-900/60 text-white placeholder:text-slate-500 focus-visible:ring-violet-500"
                />
              </div>
            )}

            <div className="flex flex-col gap-1.5">
              <Label htmlFor="email" className="text-sm font-medium text-slate-300">
                Email
              </Label>
              <Input
                id="email"
                type="email"
                placeholder="you@example.com"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                required
                className="border-slate-700 bg-slate-900/60 text-white placeholder:text-slate-500 focus-visible:ring-violet-500"
              />
            </div>

            <div className="flex flex-col gap-1.5">
              <Label htmlFor="password" className="text-sm font-medium text-slate-300">
                Password
              </Label>
              <div className="relative">
                <Input
                  id="password"
                  type={showPassword ? "text" : "password"}
                  placeholder={mode === "register" ? "Min. 8 characters" : "••••••••"}
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  required
                  minLength={mode === "register" ? 8 : 1}
                  className="border-slate-700 bg-slate-900/60 pr-10 text-white placeholder:text-slate-500 focus-visible:ring-violet-500"
                />
                <button
                  type="button"
                  onClick={() => setShowPassword((v) => !v)}
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-slate-500 hover:text-slate-300 transition-colors"
                >
                  {showPassword ? (
                    <EyeOff className="h-4 w-4" />
                  ) : (
                    <Eye className="h-4 w-4" />
                  )}
                </button>
              </div>
            </div>

            {error && (
              <div className="rounded-lg bg-red-500/10 px-3 py-2.5 text-sm text-red-400 ring-1 ring-red-500/20">
                {error}
              </div>
            )}

            <Button
              type="submit"
              disabled={loading}
              className="mt-1 w-full bg-gradient-to-r from-violet-600 to-indigo-600 font-medium text-white shadow-lg shadow-violet-500/25 hover:from-violet-500 hover:to-indigo-500"
            >
              {loading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              {mode === "login" ? "Sign In" : "Create Account"}
            </Button>

            <p className="text-center text-sm text-slate-400">
              {mode === "login" ? (
                <>
                  Don&apos;t have an account?{" "}
                  <button
                    type="button"
                    className="text-violet-400 hover:text-violet-300 underline-offset-4 hover:underline transition-colors"
                    onClick={() => setMode("register")}
                  >
                    Sign up
                  </button>
                </>
              ) : (
                <>
                  Already have an account?{" "}
                  <button
                    type="button"
                    className="text-violet-400 hover:text-violet-300 underline-offset-4 hover:underline transition-colors"
                    onClick={() => setMode("login")}
                  >
                    Sign in
                  </button>
                </>
              )}
            </p>
          </form>
        </div>
      </div>
    </div>
  );
}
