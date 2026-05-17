"use client";

import { Sidebar } from "@/components/layout/Sidebar";
import { Providers } from "@/app/providers";

export default function DashboardLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <Providers>
      <div className="flex h-screen overflow-hidden">
        <Sidebar />
        <main className="flex flex-1 flex-col overflow-y-auto">{children}</main>
      </div>
    </Providers>
  );
}
