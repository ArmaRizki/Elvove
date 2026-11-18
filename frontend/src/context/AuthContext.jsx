import { createContext, useContext, useState } from "react";
import { adminApi } from "../services/adminApi";

const AuthContext = createContext(null);

export function AuthProvider({ children }) {
  const [admin, setAdmin] = useState(() => {
    // A value that is not JSON (an old version, a manual edit) must not blank
    // the whole site: the parse failure is swallowed and the key removed.
    try {
      const stored = localStorage.getItem("admin_user");
      return stored ? JSON.parse(stored) : null;
    } catch {
      localStorage.removeItem("admin_user");
      localStorage.removeItem("admin_token");
      return null;
    }
  });

  const login = async (username, password) => {
    const response = await adminApi.login({ username, password });
    const { token, username: name, email } = response.data;
    localStorage.setItem("admin_token", token);
    localStorage.setItem("admin_user", JSON.stringify({ username: name, email }));
    setAdmin({ username: name, email });
    return response.data;
  };

  const logout = () => {
    localStorage.removeItem("admin_token");
    localStorage.removeItem("admin_user");
    setAdmin(null);
  };

  const isAuthenticated = !!admin && !!localStorage.getItem("admin_token");

  return (
    <AuthContext.Provider value={{ admin, login, logout, isAuthenticated }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used within AuthProvider");
  return ctx;
}
