import { apiClient } from "./client";
import type { Role, User } from "@/lib/types/api.types";

export interface UserListResponse {
  data: User[];
  total: number;
}

export const adminApi = {
  listUsers: async (params?: {
    page?: number;
    per_page?: number;
  }): Promise<UserListResponse> => {
    const { data } = await apiClient.get<UserListResponse>("/api/v1/admin/users", {
      params,
    });
    return data;
  },

  updateRole: async (id: string, role: Role): Promise<User> => {
    const { data } = await apiClient.put<{ data: User }>(
      `/api/v1/admin/users/${id}/role`,
      { role }
    );
    return data.data;
  },
};
