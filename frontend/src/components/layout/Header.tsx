"use client";

import { useAuthStore } from "@/lib/store/auth.store";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Bell } from "lucide-react";
import { Button } from "@/components/ui/button";

interface HeaderProps {
  title: string;
  description?: string;
}

export function Header({ title, description }: HeaderProps) {
  const user = useAuthStore((s) => s.user);

  return (
    <header className="sticky top-0 z-10 flex h-16 shrink-0 items-center justify-between border-b border-slate-200 bg-white/90 px-6 backdrop-blur-sm">
      <div>
        <h1 className="text-xl font-bold leading-tight text-slate-900">{title}</h1>
        {description && (
          <p className="mt-0.5 text-sm leading-none text-muted-foreground">
            {description}
          </p>
        )}
      </div>

      <div className="flex items-center gap-2">
        <Button
          variant="ghost"
          size="icon"
          className="h-9 w-9 text-slate-400 hover:text-slate-700"
        >
          <Bell className="h-4 w-4" />
        </Button>

        <div className="flex cursor-default items-center gap-2 rounded-full border border-slate-200 bg-slate-50 py-1 pl-2 pr-3 transition-colors hover:bg-slate-100">
          <Avatar className="h-6 w-6">
            <AvatarFallback className="bg-gradient-to-br from-violet-500 to-indigo-600 text-xs font-bold text-white">
              {user?.name?.charAt(0).toUpperCase() ?? "U"}
            </AvatarFallback>
          </Avatar>
          <span className="text-sm font-medium text-slate-700">
            {user?.name || user?.email?.split("@")[0]}
          </span>
        </div>
      </div>
    </header>
  );
}
