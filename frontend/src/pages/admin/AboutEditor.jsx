import { useEffect, useState } from "react";
import { adminApi } from "../../services/adminApi";
import LoadingState from "../../components/LoadingState";
import AdminPageHeader from "../../components/admin/AdminPageHeader";
import ImageUploadField from "../../components/admin/ImageUploadField";
import {
  adminCard,
  adminInput,
  adminLabel,
  adminPrimaryButton,
} from "../../components/admin/adminStyles";

/**
 * The About page keeps two editable elements: the story and the photo. The full
 * record still loads and is saved back unchanged, so fields the
 * public page no longer shows (introduction, years of experience, style,
 * achievements) keep their stored values instead of being wiped.
 */
export default function AboutEditor() {
  const [form, setForm] = useState(null);
  const [saving, setSaving] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [notice, setNotice] = useState({ type: "", text: "" });

  useEffect(() => {
    adminApi
      .getAbout()
      .then((res) => setForm(res.data))
      .catch(() => setNotice({ type: "error", text: "Gagal memuat halaman Tentang Kami." }));
  }, []);

  const handleSubmit = async (e) => {
    e.preventDefault();
    setSaving(true);
    setNotice({ type: "", text: "" });
    try {
      await adminApi.updateAbout(form);
      setNotice({ type: "success", text: "Halaman Tentang Kami berhasil disimpan." });
    } catch (err) {
      setNotice({
        type: "error",
        text: err.response?.data?.message || "Gagal menyimpan halaman Tentang Kami.",
      });
    } finally {
      setSaving(false);
    }
  };

  if (!form) {
    return notice.text ? (
      <p className="border border-red-200 bg-red-50 p-3 text-sm text-red-700">{notice.text}</p>
    ) : (
      <LoadingState label="Memuat halaman Tentang Kami" />
    );
  }

  return (
    <div className="min-w-0">
      <AdminPageHeader
        title="Tentang Kami"
        description="Cerita dan foto yang tampil pada halaman Tentang Kami publik."
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

      <form onSubmit={handleSubmit} className={`${adminCard} space-y-6`}>
        <div>
          <label className={adminLabel} htmlFor="about-story">CERITA</label>
          <textarea
            id="about-story"
            rows={8}
            value={form.story || ""}
            onChange={(e) => setForm({ ...form, story: e.target.value })}
            className={adminInput}
            placeholder="Tulis cerita Elvove di sini."
          />
          <p className="mt-2 text-xs text-gray-500">
            Cerita tampil paling atas pada halaman Tentang Kami.
          </p>
        </div>

        <ImageUploadField
          id="about-photo"
          label="FOTO TENTANG KAMI"
          value={form.imageUrl}
          onChange={(url) => setForm((f) => ({ ...f, imageUrl: url }))}
          onError={(text) => setNotice({ type: "error", text })}
          onUploadingChange={setUploading}
          previewClassName="h-24 w-24"
        />

        <div>
          <button
            type="submit"
            disabled={saving || uploading}
            className={`${adminPrimaryButton} w-full sm:w-auto`}
          >
            {saving ? "Menyimpan..." : "Simpan Perubahan"}
          </button>
        </div>
      </form>
    </div>
  );
}
