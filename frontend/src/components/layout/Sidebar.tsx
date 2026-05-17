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
} from "lucide-react";
import { cn } from "@/lib/utils";
import { authApi } from "@/lib/api/auth";
import { Button } from "@/components/ui/button";

const navItems = [
  { href: "/",          label: "Dashboard",   icon: LayoutDashboard },
  { href: "/channels",  label: "Channels",    icon: Tv2 },
  { href: "/content",   label: "Content",     icon: FileText },
  { href: "/videos",    label: "Videos",      icon: Video },
  { href: "/schedule",  label: "Schedule",    icon: CalendarDays },
  { href: "/products",  label: "Products",    icon: ShoppingBag },
  { href: "/analytics", label: "Analytics",   icon: BarChart3 },
  { href: "/settings",  label: "Settings",    icon: Settings },
];

export function Sidebar() {
  const pathname = usePathname();

  return (
    <aside className="flex h-screen w-60 flex-col border-r bg-background px-3 py-4">
      {/* Logo */}
      <div className="mb-6 flex items-center gap-2 px-2">
        <Zap className="h-6 w-6 text-primary" />
        <span className="text-lg font-bold">AutoContent</span>
      </div>

      {/* Nav */}
      <nav className="flex flex-1 flex-col gap-1">
        {navItems.map(({ href, label, icon: Icon }) => (
          <Link key={href} href={href}>
            <span
              className={cn(
                "flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors hover:bg-accent hover:text-accent-foreground",
                pathname === href
                  ? "bg-accent text-accent-foreground"
                  : "text-muted-foreground"
              )}
            >
              <Icon className="h-4 w-4" />
              {label}
            </span>
          </Link>
        ))}
      </nav>

      {/* Logout */}
      <Button
        variant="ghost"
        className="w-full justify-start gap-3 text-muted-foreground"
        onClick={() => authApi.logout()}
      >
        <LogOut className="h-4 w-4" />
        Logout
      </Button>
    </aside>
  );
}
