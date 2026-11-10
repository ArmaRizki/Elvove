import { useEffect, useState } from "react";
import { adminApi } from "../../services/adminApi";
import LoadingState from "../../components/LoadingState";
import AdminPageHeader from "../../components/admin/AdminPageHeader";
import {
  adminCard,
  adminInput,
  adminLabel,
  adminPrimaryButton,
} from "../../components/admin/adminStyles";

const fields = [
  "instagram",
  "threads",
  "tiktok",
  "facebook",
  "youtube",
  "pinterest",
  "linkedin",
];

const labels = {
  instagram: "Instagram",
  threads: "Threads",
  tiktok: "TikTok",
  facebook: "Facebook",
  youtube: "YouTube",
  pinterest: "Pinterest",
  linkedin: "LinkedIn",
};

const placeholders = {
  instagram: "https://www.instagram.com/...",
  threads: "https://www.threads.net/...",
  tiktok: "https://www.tiktok.com/...",
  facebook: "https://www.facebook.com/...",
  youtube: "https://www.youtube.com/...",
  pinterest: "https://www.pinterest.com/...",
  linkedin: "https://www.linkedin.com/...",
};

export default function SocialLinksEditor() {
  const [form, setForm] = useState(null);
  const [saving, setSaving] = useState(false);
  const [notice, setNotice] = useState({ type: "", text: "" });

  useEffect(() => {
    adminApi
      .getSocialLinks()
      .then((res) => setForm(res.data))
      .catch(() => setNotice({ type: "error", text: "Gagal memuat tautan media sosial." }));
  }, []);

  const handleSubmit = async (e) => {
    e.preventDefault();
    setSaving(true);
    setNotice({ type: "", text: "" });

    try {
      await adminApi.updateSocialLinks(form);
      setNotice({ type: "success", text: "Tautan media sosial berhasil disimpan." });
    } catch (err) {
      setNotice({
        type: "error",
        text: err.response?.data?.message || "Gagal menyimpan tautan media sosial.",
      });
    } finally {
      setSaving(false);
    }
  };

  if (!form) {
    return notice.text ? (
      <p className="border border-red-200 bg-red-50 p-3 text-sm text-red-700">{notice.text}</p>
    ) : (
      <LoadingState label="Memuat tautan media sosial" />
    );
  }

  return (
    <div className="min-w-0">
      <AdminPageHeader
        title="Tautan Media Sosial"
        description="URL profil yang ditautkan dari situs publik."
      />

      {notice.text && (
        <p
          className={`mb-6 border p-3 text-sm ${
            notice.type === "success"
              ? "border-green-200 bg-green-50 text-green-800"
              : "border-red-200 bg-red-50 text-red-700"
          }`}
          role="status"
        >
          {notice.text}
        </p>
      )}

      <form onSubmit={handleSubmit} className={`${adminCard} space-y-5`}>
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          {fields.map((field) => (
            <div key={field}>
              <label className={adminLabel} htmlFor={`social-${field}`}>
                {labels[field].toUpperCase()} URL
              </label>
              <input
                id={`social-${field}`}
                type="url"
                value={form[field] || ""}
                placeholder={placeholders[field]}
                onChange={(e) => setForm({ ...form, [field]: e.target.value })}
                className={adminInput}
              />
            </div>
          ))}
        </div>

        <div>
          <button
            type="submit"
            disabled={saving}
            className={`${adminPrimaryButton} w-full sm:w-auto`}
          >
            {saving ? "Menyimpan..." : "Simpan Perubahan"}
          </button>
        </div>
      </form>
    </div>
  );
}
