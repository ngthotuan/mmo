import { apiClient } from "./client";
import type { ContentPlan, TrendTopic, ListResponse } from "@/lib/types/api.types";

export const contentApi = {
  listTrends: async (params?: { status?: string; page?: number }): Promise<ListResponse<TrendTopic>> => {
    const { data } = await apiClient.get("/api/v1/trends", { params });
    return data;
  },

  discoverTrends: async (): Promise<void> => {
    await apiClient.post("/api/v1/trends/discover");
  },

  listPlans: async (params?: { status?: string; page?: number }): Promise<ListResponse<ContentPlan>> => {
    const { data } = await apiClient.get("/api/v1/content", { params });
    return data;
  },

  getPlan: async (id: string): Promise<ContentPlan> => {
    const { data } = await apiClient.get<ContentPlan>(`/api/v1/content/${id}`);
    return data;
  },

  createFromTrend: async (params: {
    topic_id: string;
    niche?: string;
    platforms?: string[];
    auto_approve?: boolean;
  }): Promise<ContentPlan> => {
    const { data } = await apiClient.post<ContentPlan>("/api/v1/content", params);
    return data;
  },

  updatePlan: async (id: string, updates: Partial<Pick<ContentPlan, "title" | "script" | "notes" | "voice">> & { niche?: string; target_platforms?: string[] }): Promise<ContentPlan> => {
    const { data } = await apiClient.put<ContentPlan>(`/api/v1/content/${id}`, updates);
    return data;
  },

  approvePlan: async (id: string): Promise<void> => {
    await apiClient.post(`/api/v1/content/${id}/approve`);
  },

  rejectPlan: async (id: string): Promise<void> => {
    await apiClient.post(`/api/v1/content/${id}/reject`);
  },

  regenerateScript: async (id: string): Promise<ContentPlan> => {
    const { data } = await apiClient.post<ContentPlan>(`/api/v1/content/${id}/generate-script`);
    return data;
  },

  deletePlan: async (id: string): Promise<void> => {
    await apiClient.delete(`/api/v1/content/${id}`);
  },

  bulkActionPlans: async (action: "approve" | "reject" | "delete", ids: string[]): Promise<{ processed: number }> => {
    const { data } = await apiClient.post("/api/v1/content/bulk-action", { action, ids });
    return data;
  },

  bulkRejectTrends: async (ids: string[]): Promise<{ processed: number }> => {
    const { data } = await apiClient.post("/api/v1/trends/bulk-reject", { ids });
    return data;
  },
};
