"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import {
  LayoutDashboard,
  Tv2,
  FileText,
  Video,
  CalendarDays,
  BarChart3,
  Settings,
  LogOut,
  Zap,
  ShoppingBag,
  Send,
} from "lucide-react";
import { cn } from "@/lib/utils";
import { authApi } from "@/lib/api/auth";
import { useAuthStore } from "@/lib/store/auth.store";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";

type NavItem = {
  href: string;
  label: string;
  icon: React.ComponentType<{ className?: string }>;
  exact?: boolean;
};

const navGroups: { label: string; items: NavItem[] }[] = [
  {
    label: "Overview",
    items: [
      { href: "/", label: "Dashboard", icon: LayoutDashboard, exact: true },
    ],
  },
  {
    label: "Pipeline",
    items: [
      { href: "/content", label: "Content", icon: FileText },
      { href: "/videos", label: "Videos", icon: Video },
      { href: "/publish", label: "Publish", icon: Send },
      { href: "/schedule", label: "Schedule", icon: CalendarDays },
    ],
  },
  {
    label: "Social",
    items: [
      { href: "/channels", label: "Channels", icon: Tv2 },
      { href: "/products", label: "Products", icon: ShoppingBag },
    ],
  },
  {
    label: "Insights",
    items: [
      { href: "/analytics", label: "Analytics", icon: BarChart3 },
    ],
  },
];

export function Sidebar() {
  const pathname = usePathname();
  const user = useAuthStore((s) => s.user);

  const isActive = (href: string, exact = false) => {
    if (exact) return pathname === href;
    return pathname === href || pathname.startsWith(href + "/");
  };

  return (
    <aside className="flex h-screen w-60 shrink-0 flex-col bg-slate-900">
      {/* Brand */}
      <div className="flex items-center gap-3 border-b border-slate-800 px-5 py-4">
        <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-gradient-to-br from-violet-500 to-indigo-600 shadow-lg shadow-violet-500/25">
          <Zap className="h-4 w-4 text-white" />
        </div>
        <div className="min-w-0">
          <p className="text-sm font-bold leading-none text-white">AutoContent</p>
          <p className="mt-0.5 text-xs leading-none text-slate-500">Automation Platform</p>
        </div>
      </div>

      {/* Navigation */}
      <nav className="flex flex-1 flex-col gap-5 overflow-y-auto px-3 py-4">
        {navGroups.map((group) => (
          <div key={group.label}>
            <p className="mb-1 px-3 text-xs font-semibold uppercase tracking-wider text-slate-600">
              {group.label}
            </p>
            <div className="flex flex-col gap-0.5">
              {group.items.map(({ href, label, icon: Icon, exact = false }) => {
                const active = isActive(href, exact);
                return (
                  <Link key={href} href={href}>
                    <span
                      className={cn(
                        "flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium transition-all duration-150",
                        active
                          ? "bg-violet-600/20 text-white ring-1 ring-violet-500/30"
                          : "text-slate-400 hover:bg-slate-800 hover:text-slate-100"
                      )}
                    >
                      <Icon
                        className={cn(
                          "h-4 w-4 shrink-0",
                          active ? "text-violet-400" : ""
                        )}
                      />
                      {label}
                      {active && (
                        <span className="ml-auto h-1.5 w-1.5 rounded-full bg-violet-400" />
                      )}
                    </span>
                  </Link>
                );
              })}
            </div>
          </div>
        ))}
      </nav>

      {/* Footer */}
      <div className="border-t border-slate-800 px-3 pb-3 pt-3">
        <Link href="/settings">
          <span
            className={cn(
              "mb-2 flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium transition-all duration-150",
              isActive("/settings")
                ? "bg-violet-600/20 text-white ring-1 ring-violet-500/30"
                : "text-slate-400 hover:bg-slate-800 hover:text-slate-100"
            )}
          >
            <Settings
              className={cn(
                "h-4 w-4 shrink-0",
                isActive("/settings") ? "text-violet-400" : ""
              )}
            />
            Settings
          </span>
        </Link>

        <div className="flex items-center gap-2.5 rounded-lg px-3 py-2">
          <Avatar className="h-7 w-7 shrink-0">
            <AvatarFallback className="bg-slate-700 text-xs font-bold text-slate-200">
              {user?.name?.charAt(0).toUpperCase() ?? "U"}
            </AvatarFallback>
          </Avatar>
          <div className="min-w-0 flex-1">
            <p className="truncate text-xs font-medium text-slate-200">
              {user?.name || user?.email?.split("@")[0]}
            </p>
            <p className="truncate text-xs text-slate-500">{user?.email}</p>
          </div>
          <button
            onClick={() => authApi.logout()}
            className="shrink-0 rounded-md p-1 text-slate-500 transition-colors hover:bg-slate-700 hover:text-slate-200"
            title="Logout"
          >
            <LogOut className="h-3.5 w-3.5" />
          </button>
        </div>
      </div>
    </aside>
  );
}
