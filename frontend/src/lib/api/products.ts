import { apiClient } from "./client";
import type { Product } from "@/lib/types/api.types";

export interface ProductListResponse {
  data: Product[];
  total: number;
}

export interface CreateProductBody {
  platform: string;
  platform_product_id: string;
  name: string;
  description?: string;
  price?: number;
  currency?: string;
  cover_image_url?: string;
  product_url?: string;
}

export interface SyncProductsBody {
  platform: "tiktok" | "facebook";
  channel_id: string;
  catalog_id?: string;
}

export const productsApi = {
  list: async (params?: { platform?: string; page?: number; perPage?: number }): Promise<ProductListResponse> => {
    const { data } = await apiClient.get<ProductListResponse>("/api/v1/products", { params });
    return data;
  },

  get: async (id: string): Promise<Product> => {
    const { data } = await apiClient.get<{ data: Product }>(`/api/v1/products/${id}`);
    return data.data;
  },

  create: async (body: CreateProductBody): Promise<Product> => {
    const { data } = await apiClient.post<{ data: Product }>("/api/v1/products", body);
    return data.data;
  },

  delete: async (id: string): Promise<void> => {
    await apiClient.delete(`/api/v1/products/${id}`);
  },

  sync: async (body: SyncProductsBody): Promise<{ synced: number }> => {
    const { data } = await apiClient.post<{ synced: number }>("/api/v1/products/sync", body);
    return data;
  },

  listByPublishJob: async (publishJobId: string): Promise<Product[]> => {
    const { data } = await apiClient.get<{ data: Product[] }>(`/api/v1/publish/${publishJobId}/products`);
    return data.data;
  },

  tagPublishJob: async (publishJobId: string, productIds: string[]): Promise<void> => {
    await apiClient.post(`/api/v1/publish/${publishJobId}/products`, { product_ids: productIds });
  },
};
