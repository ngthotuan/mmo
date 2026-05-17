"use client";

import { useAuthStore } from "@/lib/store/auth.store";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";

interface HeaderProps {
  title: string;
}

export function Header({ title }: HeaderProps) {
  const user = useAuthStore((s) => s.user);

  return (
    <header className="flex h-14 items-center justify-between border-b px-6">
      <h1 className="text-lg font-semibold">{title}</h1>
      <div className="flex items-center gap-3">
        <span className="text-sm text-muted-foreground">{user?.email}</span>
        <Avatar className="h-8 w-8">
          <AvatarFallback className="text-xs">
            {user?.name?.charAt(0).toUpperCase() ?? "U"}
          </AvatarFallback>
        </Avatar>
      </div>
    </header>
  );
}
