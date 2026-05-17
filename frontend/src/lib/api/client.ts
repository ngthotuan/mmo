import axios, { type AxiosInstance, type AxiosError } from "axios";

const API_URL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

const ACCESS_TOKEN_KEY = "mmo_access_token";
const REFRESH_TOKEN_KEY = "mmo_refresh_token";

export const getAccessToken = () =>
  typeof window !== "undefined" ? localStorage.getItem(ACCESS_TOKEN_KEY) : null;

export const setTokens = (access: string, refresh: string) => {
  localStorage.setItem(ACCESS_TOKEN_KEY, access);
  localStorage.setItem(REFRESH_TOKEN_KEY, refresh);
};

export const clearTokens = () => {
  localStorage.removeItem(ACCESS_TOKEN_KEY);
  localStorage.removeItem(REFRESH_TOKEN_KEY);
};

export const apiClient: AxiosInstance = axios.create({
  baseURL: API_URL,
  headers: { "Content-Type": "application/json" },
});

// Attach access token to every request
apiClient.interceptors.request.use((config) => {
  const token = getAccessToken();
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

// Auto-refresh on 401
let refreshing = false;
apiClient.interceptors.response.use(
  (res) => res,
  async (error: AxiosError) => {
    const original = error.config as typeof error.config & { _retry?: boolean };
    if (error.response?.status === 401 && !original?._retry && !refreshing) {
      original._retry = true;
      refreshing = true;
      try {
        const refresh = localStorage.getItem(REFRESH_TOKEN_KEY);
        if (!refresh) throw new Error("no refresh token");
        const { data } = await axios.post(`${API_URL}/api/v1/auth/refresh`, {
          refresh_token: refresh,
        });
        setTokens(data.access_token, data.refresh_token);
        if (original) {
          original.headers!.Authorization = `Bearer ${data.access_token}`;
          return apiClient(original);
        }
      } catch {
        clearTokens();
        window.location.href = "/login";
      } finally {
        refreshing = false;
      }
    }
    return Promise.reject(error);
  }
);
