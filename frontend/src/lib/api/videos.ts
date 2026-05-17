import { apiClient } from "./client";
import type { VideoJob } from "@/lib/types/api.types";

export const videosApi = {
  list: async (params?: { status?: string; page?: number; per_page?: number }) => {
    const { data } = await apiClient.get<{ data: VideoJob[]; total: number }>("/api/v1/videos", { params });
    return data;
  },

  get: async (id: string): Promise<VideoJob> => {
    const { data } = await apiClient.get<{ data: VideoJob }>(`/api/v1/videos/${id}`);
    return data.data;
  },

  retry: async (id: string): Promise<void> => {
    await apiClient.post(`/api/v1/videos/${id}/retry`);
  },

  delete: async (id: string): Promise<void> => {
    await apiClient.delete(`/api/v1/videos/${id}`);
  },

  getDownloadURL: async (id: string): Promise<string> => {
    const { data } = await apiClient.get<{ url: string }>(`/api/v1/videos/${id}/download`);
    return data.url;
  },
};
