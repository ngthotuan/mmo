import { apiClient } from "./client";
import type { Channel, ListResponse } from "@/lib/types/api.types";

export const channelsApi = {
  list: async (): Promise<Channel[]> => {
    const { data } = await apiClient.get<{ data: Channel[] }>("/api/v1/channels");
    return data.data;
  },

  getAuthURL: async (platform: "tiktok" | "facebook" | "youtube"): Promise<string> => {
    const { data } = await apiClient.get<{ auth_url: string }>(
      `/api/v1/channels/connect/${platform}`
    );
    return data.auth_url;
  },

  connectTikTok: async (code: string, state: string): Promise<Channel> => {
    const { data } = await apiClient.post<Channel>("/api/v1/channels/oauth/tiktok", { code, state });
    return data;
  },

  connectYouTube: async (code: string, state: string): Promise<Channel> => {
    const { data } = await apiClient.post<Channel>("/api/v1/channels/oauth/youtube", { code, state });
    return data;
  },

  connectFacebook: async (userToken: string, pageId: string): Promise<Channel> => {
    const { data } = await apiClient.post<Channel>("/api/v1/channels/oauth/facebook", {
      user_token: userToken,
      page_id: pageId,
    });
    return data;
  },

  getFacebookPages: async (code: string): Promise<{ pages: { id: string; name: string }[]; userToken: string }> => {
    const { data } = await apiClient.get<{
      data: { id: string; name: string }[];
      user_token: string;
    }>(`/api/v1/channels/facebook/pages?code=${encodeURIComponent(code)}`);
    return { pages: data.data, userToken: data.user_token };
  },

  delete: async (id: string): Promise<void> => {
    await apiClient.delete(`/api/v1/channels/${id}`);
  },

  toggle: async (id: string, active: boolean): Promise<void> => {
    await apiClient.put(`/api/v1/channels/${id}/toggle`, { active });
  },
};
