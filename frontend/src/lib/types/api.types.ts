export type Platform = "tiktok" | "facebook";

export type ContentStatus =
  | "draft"
  | "approved"
  | "rejected"
  | "video_queued"
  | "video_ready"
  | "scheduled"
  | "published";

export type VideoJobStatus =
  | "pending"
  | "media_collecting"
  | "tts_generating"
  | "assembling"
  | "uploading"
  | "done"
  | "failed";

export type PublishJobStatus =
  | "scheduled"
  | "publishing"
  | "published"
  | "failed"
  | "cancelled";

export interface User {
  id: string;
  email: string;
  name: string;
  role: string;
  created_at: string;
}

export interface Channel {
  id: string;
  user_id: string;
  platform: Platform;
  platform_user_id: string;
  username: string;
  display_name: string;
  avatar_url: string;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface TrendTopic {
  id: string;
  source?: string;
  title: string;
  description?: string;
  keywords: string[];
  trending_score: number;
  source_url?: string;
  status: string;
  discovered_at: string;
}

export interface ContentPlan {
  id: string;
  title: string;
  niche: string;
  target_platforms: Platform[];
  script: string;
  script_metadata: {
    hook: string;
    cta: string;
    hashtags: string[];
    caption: string;
  };
  status: ContentStatus;
  auto_approve: boolean;
  voice: string;
  notes: string;
  created_at: string;
  updated_at: string;
}

export interface VideoJob {
  id: string;
  content_plan_id: string;
  status: VideoJobStatus;
  output_video_url: string;
  duration_seconds: number;
  file_size_bytes: number;
  retry_count: number;
  error_message: string;
  created_at: string;
  updated_at: string;
}

export interface PublishJob {
  id: string;
  video_job_id: string;
  channel_id: string;
  platform: Platform;
  caption: string;
  hashtags: string[];
  scheduled_at: string;
  published_at: string;
  platform_post_url: string;
  status: PublishJobStatus;
  error_message: string;
  created_at: string;
}

export interface Pagination {
  page: number;
  per_page: number;
  total: number;
}

export interface ListResponse<T> {
  data: T[];
  pagination: Pagination;
}

export interface TokenResponse {
  access_token: string;
  refresh_token: string;
  expires_in: number;
}

export interface Product {
  id: string;
  user_id: string;
  channel_id?: string;
  platform: Platform;
  platform_product_id: string;
  name: string;
  description: string;
  price: number;
  currency: string;
  cover_image_url: string;
  product_url: string;
  status: string;
  synced_at: string;
  created_at: string;
  updated_at: string;
}
