import { apiClient } from "./client";

export interface PipelineStatus {
  video_status_counts: Record<string, number>;
  active_jobs: Array<{ id: string; status: string; created: string }>;
  total_videos: number;
}

export const pipelineApi = {
  status: async (): Promise<PipelineStatus> => {
    const { data } = await apiClient.get<PipelineStatus>("/api/v1/pipeline/status");
    return data;
  },
};
