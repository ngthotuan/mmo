import { apiClient } from "./client";
import type { PublishJob } from "@/lib/types/api.types";

export interface CreatePublishRequest {
  video_job_id: string;
  channel_id: string;
  caption: string;
  hashtags: string[];
  scheduled_at?: string;
}

export const publishApi = {
  list: async (params?: { status?: string; page?: number }) => {
    const { data } = await apiClient.get<{ data: PublishJob[]; total: number }>("/api/v1/publish", { params });
    return data;
  },

  create: async (req: CreatePublishRequest): Promise<PublishJob> => {
    const { data } = await apiClient.post<{ data: PublishJob }>("/api/v1/publish", req);
    return data.data;
  },

  get: async (id: string): Promise<PublishJob> => {
    const { data } = await apiClient.get<{ data: PublishJob }>(`/api/v1/publish/${id}`);
    return data.data;
  },

  update: async (id: string, body: { caption?: string; hashtags?: string[]; scheduled_at?: string }): Promise<PublishJob> => {
    const { data } = await apiClient.put<{ data: PublishJob }>(`/api/v1/publish/${id}`, body);
    return data.data;
  },

  cancel: async (id: string): Promise<void> => {
    await apiClient.delete(`/api/v1/publish/${id}`);
  },

  publishNow: async (id: string): Promise<void> => {
    await apiClient.post(`/api/v1/publish/${id}/publish-now`);
  },

  calendar: async (start: string, end: string): Promise<PublishJob[]> => {
    const { data } = await apiClient.get<{ data: PublishJob[] }>("/api/v1/calendar", {
      params: { start, end },
    });
    return data.data;
  },
};
