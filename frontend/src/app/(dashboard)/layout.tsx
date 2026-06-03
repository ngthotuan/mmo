"use client";

import { Eye } from "lucide-react";
import { Sidebar } from "@/components/layout/Sidebar";
import { Providers } from "@/app/providers";
import { usePipelineSSE } from "@/hooks/usePipelineSSE";
import { useAuth } from "@/lib/hooks/useAuth";
import { hasFullAccess } from "@/lib/types/api.types";

function DashboardShell({ children }: { children: React.ReactNode }) {
  usePipelineSSE();
  // Hydrate the auth store (current user + role) for the whole dashboard.
  const { user } = useAuth();
  const viewer = !!user && !hasFullAccess(user.role);

  return (
    <div className="flex h-screen overflow-hidden">
      <Sidebar />
      <main className="flex flex-1 flex-col overflow-y-auto bg-slate-50">
        {viewer && (
          <div className="flex items-center gap-2 border-b border-amber-200 bg-amber-50 px-6 py-2 text-sm text-amber-800">
            <Eye className="h-4 w-4 shrink-0" />
            <span>
              Your account has <strong>view-only</strong> access. Ask an admin to
              grant full access to create, edit, or publish.
            </span>
          </div>
        )}
        {children}
      </main>
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
