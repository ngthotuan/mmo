"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { ShieldAlert, Users as UsersIcon } from "lucide-react";
import { toast } from "sonner";
import { Header } from "@/components/layout/Header";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { adminApi } from "@/lib/api/admin";
import { useAuthStore } from "@/lib/store/auth.store";
import type { Role, User } from "@/lib/types/api.types";

const ROLES: { value: Role; label: string; hint: string }[] = [
  { value: "admin", label: "Admin", hint: "Full access + user management" },
  { value: "member", label: "Member", hint: "Full access to all features" },
  { value: "viewer", label: "Viewer", hint: "Read-only" },
];

const roleBadgeVariant = (role: Role) =>
  role === "admin" ? "default" : role === "member" ? "secondary" : "outline";

export default function AdminUsersPage() {
  const currentUser = useAuthStore((s) => s.user);
  const queryClient = useQueryClient();

  const { data, isLoading } = useQuery({
    queryKey: ["admin", "users"],
    queryFn: () => adminApi.listUsers({ per_page: 100 }),
  });

  const mutation = useMutation({
    mutationFn: ({ id, role }: { id: string; role: Role }) =>
      adminApi.updateRole(id, role),
    onSuccess: (updated) => {
      toast.success(`Updated ${updated.email} to ${updated.role}`);
      queryClient.invalidateQueries({ queryKey: ["admin", "users"] });
    },
    onError: (err: unknown) => {
      const msg =
        (err as { response?: { data?: { message?: string } } })?.response?.data
          ?.message ?? "Failed to update role";
      toast.error(msg);
    },
  });

  // Guard: only admins may see this page.
  if (currentUser && currentUser.role !== "admin") {
    return (
      <>
        <Header title="Users" />
        <div className="flex flex-1 flex-col items-center justify-center gap-2 text-muted-foreground">
          <ShieldAlert className="h-8 w-8" />
          <p>You need an admin account to manage users.</p>
        </div>
      </>
    );
  }

  return (
    <>
      <Header
        title="Users"
        description="Manage accounts and grant access. New sign-ups start as view-only."
      />
      <div className="p-6">
        <div className="rounded-lg border bg-card">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Name</TableHead>
                <TableHead>Email</TableHead>
                <TableHead>Joined</TableHead>
                <TableHead>Role</TableHead>
                <TableHead className="text-right">Set role</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {isLoading &&
                Array.from({ length: 4 }).map((_, i) => (
                  <TableRow key={i}>
                    <TableCell colSpan={5}>
                      <Skeleton className="h-6 w-full" />
                    </TableCell>
                  </TableRow>
                ))}

              {!isLoading && (data?.data.length ?? 0) === 0 && (
                <TableRow>
                  <TableCell colSpan={5} className="text-center text-muted-foreground">
                    <div className="flex flex-col items-center gap-2 py-8">
                      <UsersIcon className="h-6 w-6" />
                      No users yet.
                    </div>
                  </TableCell>
                </TableRow>
              )}

              {data?.data.map((u: User) => {
                const isSelf = u.id === currentUser?.id;
                return (
                  <TableRow key={u.id}>
                    <TableCell className="font-medium">{u.name}</TableCell>
                    <TableCell className="text-muted-foreground">{u.email}</TableCell>
                    <TableCell className="text-muted-foreground">
                      {new Date(u.created_at).toLocaleDateString()}
                    </TableCell>
                    <TableCell>
                      <Badge variant={roleBadgeVariant(u.role)} className="capitalize">
                        {u.role}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-right">
                      <select
                        value={u.role}
                        disabled={isSelf || mutation.isPending}
                        title={isSelf ? "You cannot change your own role" : undefined}
                        onChange={(e) =>
                          mutation.mutate({ id: u.id, role: e.target.value as Role })
                        }
                        className="rounded-md border border-input bg-background px-2 py-1 text-sm disabled:cursor-not-allowed disabled:opacity-50"
                      >
                        {ROLES.map((r) => (
                          <option key={r.value} value={r.value}>
                            {r.label}
                          </option>
                        ))}
                      </select>
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        </div>
        <p className="mt-3 text-xs text-muted-foreground">
          Roles — <strong>Admin</strong>: full access + user management.{" "}
          <strong>Member</strong>: full access. <strong>Viewer</strong>: read-only
          (default for new sign-ups).
        </p>
      </div>
    </>
  );
}
