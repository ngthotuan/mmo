"use client";

import { Sidebar } from "@/components/layout/Sidebar";
import { Providers } from "@/app/providers";
import { usePipelineSSE } from "@/hooks/usePipelineSSE";

function DashboardShell({ children }: { children: React.ReactNode }) {
  usePipelineSSE();
  return (
    <div className="flex h-screen overflow-hidden">
      <Sidebar />
      <main className="flex flex-1 flex-col overflow-y-auto bg-slate-50">{children}</main>
    </div>
  );
}

export default function DashboardLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <Providers>
      <DashboardShell>{children}</DashboardShell>
    </Providers>
  );
}
