import { apiClient, setTokens, clearTokens } from "./client";
import type { User, TokenResponse } from "@/lib/types/api.types";

export const authApi = {
  login: async (email: string, password: string): Promise<TokenResponse> => {
    const { data } = await apiClient.post<TokenResponse>("/api/v1/auth/login", {
      email,
      password,
    });
    setTokens(data.access_token, data.refresh_token);
    return data;
  },

  register: async (
    name: string,
    email: string,
    password: string
  ): Promise<TokenResponse> => {
    const { data } = await apiClient.post<TokenResponse>(
      "/api/v1/auth/register",
      { name, email, password }
    );
    setTokens(data.access_token, data.refresh_token);
    return data;
  },

  logout: () => {
    clearTokens();
    window.location.href = "/login";
  },

  me: async (): Promise<User> => {
    const { data } = await apiClient.get<User>("/api/v1/auth/me");
    return data;
  },

  updateProfile: async (name: string): Promise<void> => {
    await apiClient.put("/api/v1/auth/profile", { name });
  },

  changePassword: async (currentPassword: string, newPassword: string): Promise<void> => {
    await apiClient.put("/api/v1/auth/change-password", {
      current_password: currentPassword,
      new_password: newPassword,
    });
  },
};
