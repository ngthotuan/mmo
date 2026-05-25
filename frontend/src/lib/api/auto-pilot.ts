import { apiClient } from "./client";

export interface AutoPilotProfile {
  id: string;
  name: string;
  niche: string;
  voice: string;
  target_platforms: string[];
  trend_filter: string;
  trend_sources: string[];
  daily_count: number;
  schedule_times: string[];
  auto_approve: boolean;
  auto_publish: boolean;
  enabled: boolean;
  last_run_at: string | null;
  last_run_count: number;
  total_videos: number;
  created_at: string;
}

export interface AutoPilotInput {
  name: string;
  niche?: string;
  voice?: string;
  target_platforms: string[];
  trend_filter?: string;
  trend_sources?: string[];
  daily_count: number;
  schedule_times: string[];
  auto_approve?: boolean;
  auto_publish?: boolean;
  enabled?: boolean;
}

export const autoPilotApi = {
  list: async (): Promise<{ data: AutoPilotProfile[]; total: number }> => {
    const { data } = await apiClient.get("/api/v1/auto-pilot");
    return data;
  },

  get: async (id: string): Promise<AutoPilotProfile> => {
    const { data } = await apiClient.get(`/api/v1/auto-pilot/${id}`);
    return data.data;
  },

  create: async (input: AutoPilotInput): Promise<AutoPilotProfile> => {
    const { data } = await apiClient.post("/api/v1/auto-pilot", input);
    return data.data;
  },

  update: async (id: string, input: AutoPilotInput): Promise<AutoPilotProfile> => {
    const { data } = await apiClient.put(`/api/v1/auto-pilot/${id}`, input);
    return data.data;
  },

  toggle: async (id: string, enabled: boolean): Promise<void> => {
    await apiClient.put(`/api/v1/auto-pilot/${id}/toggle`, { enabled });
  },

  delete: async (id: string): Promise<void> => {
    await apiClient.delete(`/api/v1/auto-pilot/${id}`);
  },

  runNow: async (id: string): Promise<{ plans_created: number }> => {
    const { data } = await apiClient.post(`/api/v1/auto-pilot/${id}/run`);
    return data;
  },
};
