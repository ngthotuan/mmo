"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { useQuery } from "@tanstack/react-query";
import { authApi } from "@/lib/api/auth";
import { useAuthStore } from "@/lib/store/auth.store";
import { getAccessToken } from "@/lib/api/client";

export function useAuth() {
  const { user, setUser, setLoading } = useAuthStore();

  const { data, isLoading, error } = useQuery({
    queryKey: ["me"],
    queryFn: authApi.me,
    enabled: !!getAccessToken(),
    retry: false,
    staleTime: 5 * 60 * 1000,
  });

  useEffect(() => {
    if (data) setUser(data);
    if (error) setUser(null);
    setLoading(isLoading);
  }, [data, isLoading, error, setUser, setLoading]);

  return { user, isLoading };
}

export function useRequireAuth() {
  const router = useRouter();
  const { user, isLoading } = useAuth();

  useEffect(() => {
    if (!isLoading && !user && !getAccessToken()) {
      router.push("/login");
    }
  }, [user, isLoading, router]);

  return { user, isLoading };
}
