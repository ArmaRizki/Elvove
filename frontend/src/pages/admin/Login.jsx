import { useState } from "react";
import { useLocation, useNavigate } from "react-router-dom";
import { useAuth } from "../../context/AuthContext";
import { Camera } from "lucide-react";

// Only in-app admin paths are accepted, so the value cannot send the admin off-site.
function safeReturnPath(from) {
  return typeof from === "string" && from.startsWith("/admin/") && !from.startsWith("/admin/login")
    ? from
    : "/admin/dashboard";
}

export default function Login() {
  const { login } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();
  const [form, setForm] = useState({ username: "", password: "" });
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  const handleSubmit = async (e) => {
    e.preventDefault();
    setError("");
    setSubmitting(true);
    try {
      await login(form.username, form.password);
      navigate(safeReturnPath(location.state?.from));
    } catch {
      setError("Nama pengguna atau kata sandi salah.");
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-white px-6">
      <div className="w-full max-w-sm">
        <div className="flex items-center gap-2 justify-center mb-8">
          <Camera size={22} className="text-[#E11D48]" strokeWidth={1.5} />
          <span className="font-display text-xl text-gray-900">ELVOVE Admin</span>
        </div>

        <form onSubmit={handleSubmit} className="space-y-4 border border-gray-200 p-8">
          <div>
            <label className="text-xs text-gray-700 tracking-widest" htmlFor="login-username">NAMA PENGGUNA</label>
            <input
              id="login-username"
              autoComplete="username"
              value={form.username}
              onChange={(e) => setForm({ ...form, username: e.target.value })}
              className="w-full mt-1 bg-transparent border border-gray-200 focus:border-[#E11D48] outline-none px-4 py-2.5 text-gray-900"
              required
            />
          </div>
          <div>
            <label className="text-xs text-gray-700 tracking-widest" htmlFor="login-password">KATA SANDI</label>
            <input
              id="login-password"
              type="password"
              autoComplete="current-password"
              value={form.password}
              onChange={(e) => setForm({ ...form, password: e.target.value })}
              className="w-full mt-1 bg-transparent border border-gray-200 focus:border-[#E11D48] outline-none px-4 py-2.5 text-gray-900"
              required
            />
          </div>
          {error && <p className="text-burgundy text-sm">{error}</p>}
          <button
            type="submit"
            disabled={submitting}
            className="w-full bg-[#E11D48] text-ink py-2.5 text-sm tracking-wide hover:bg-[#BE123C] transition-colors disabled:opacity-50"
          >
            {submitting ? "Memproses..." : "Masuk"}
          </button>
        </form>
      </div>
    </div>
  );
}
