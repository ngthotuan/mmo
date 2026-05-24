import { apiClient } from "./client";

export interface OverviewStats {
  total_views: number;
  total_likes: number;
  total_comments: number;
  total_shares: number;
  post_count: number;
}

export interface PostAnalyticsSummary {
  publish_job_id: string;
  platform: string;
  synced_at: string;
  views: number;
  likes: number;
  comments: number;
  shares: number;
}

export interface TimeseriesPoint {
  date: string;
  views: number;
  likes: number;
  comments: number;
}

export const analyticsApi = {
  overview: async (days = 30): Promise<{ data: OverviewStats; days: number }> => {
    const { data } = await apiClient.get("/api/v1/analytics/overview", { params: { days } });
    return data;
  },

  listPosts: async (params?: { page?: number; per_page?: number }): Promise<{ data: PostAnalyticsSummary[]; total: number }> => {
    const { data } = await apiClient.get("/api/v1/analytics/posts", { params });
    return data;
  },

  timeseries: async (days = 30): Promise<{ data: TimeseriesPoint[]; days: number }> => {
    const { data } = await apiClient.get("/api/v1/analytics/timeseries", { params: { days } });
    return data;
  },
};
