import axios from "axios";

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || "";

const api = axios.create({
  baseURL: API_BASE_URL,
  headers: { "Content-Type": "application/json" },
});

// Attach the admin JWT (if present) to every outgoing request
api.interceptors.request.use((config) => {
  const token = localStorage.getItem("admin_token");
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

// On 401 (expired/invalid token), clear session and bounce to login
api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401 && window.location.pathname.startsWith("/admin") && window.location.pathname !== "/admin/login") {
      localStorage.removeItem("admin_token");
      localStorage.removeItem("admin_user");
      window.location.href = "/admin/login";
    }
    return Promise.reject(error);
  }
);

export default api;
