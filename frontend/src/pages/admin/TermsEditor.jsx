import { useEffect, useState } from "react";
import { adminApi } from "../../services/adminApi";
import LoadingState from "../../components/LoadingState";
import AdminPageHeader from "../../components/admin/AdminPageHeader";
import {
  adminCard,
  adminInput,
  adminLabel,
  adminNotice,
  adminPrimaryButton,
} from "../../components/admin/adminStyles";

export default function TermsEditor() {
  const [form, setForm] = useState({
    title: "",
    content: "",
    version: "1",
  });
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [notice, setNotice] = useState({ type: "", text: "" });

  useEffect(() => {
    loadTerms();
  }, []);

  async function loadTerms() {
    try {
      const { data } = await adminApi.getTerms();
      setForm(data);
    } catch (error) {
      setNotice({
        type: "error",
        text: error.response?.data?.message || "Gagal memuat Syarat dan Ketentuan.",
      });
    } finally {
      setLoading(false);
    }
  }

  function handleChange(event) {
    setForm((current) => ({
      ...current,
      [event.target.name]: event.target.value,
    }));
  }

  async function handleSave() {
    if (!form.title.trim() || !form.content.trim()) {
      setNotice({ type: "error", text: "Judul dan isi wajib diisi." });
      return;
    }

    setSaving(true);
    setNotice({ type: "", text: "" });

    try {
      // The introduction is no longer edited here. It is not sent, so the
      // stored value is left exactly as it was.
      const { data } = await adminApi.updateTerms({
        title: form.title,
        content: form.content,
      });
      setForm(data);
      setNotice({ type: "success", text: "Syarat dan Ketentuan berhasil disimpan dan diterbitkan." });
    } catch (error) {
      setNotice({
        type: "error",
        text: error.response?.data?.message || "Gagal menyimpan Syarat dan Ketentuan.",
      });
    } finally {
      setSaving(false);
    }
  }

  if (loading) return <LoadingState label="Memuat Syarat dan Ketentuan" />;

  return (
    <div className="min-w-0">
      <AdminPageHeader
        title="Syarat dan Ketentuan"
        description="Ubah syarat dan ketentuan yang ditampilkan pada situs publik."
      />

      {notice.text && (
        <div className={`mb-6 ${adminNotice(notice.type)}`} role="status">
          {notice.text}
        </div>
      )}

      <div className={`${adminCard} space-y-5`}>
        <div>
          <label className={adminLabel} htmlFor="terms-title">JUDUL HALAMAN</label>
          <input
            id="terms-title"
            name="title"
            value={form.title}
            onChange={handleChange}
            className={adminInput}
          />
        </div>

        <div>
          <label className={adminLabel} htmlFor="terms-content">ISI SYARAT DAN KETENTUAN (MARKDOWN)</label>
          <p className="mb-2 mt-1 text-xs text-gray-500">
            # Judul, ## Subjudul, **tebal**, - daftar butir, 1. daftar bernomor.
          </p>
          <textarea
            id="terms-content"
            name="content"
            value={form.content}
            onChange={handleChange}
            rows={18}
            spellCheck={false}
            className={`${adminInput} font-mono text-xs leading-relaxed sm:text-sm`}
          />
        </div>

        <div className="flex flex-col gap-2 sm:flex-row">
          <button
            type="button"
            onClick={handleSave}
            disabled={saving}
            className={`${adminPrimaryButton} w-full sm:w-auto`}
          >
            {saving ? "Menyimpan..." : "Simpan & Terbitkan"}
          </button>
        </div>
      </div>
    </div>
  );
}
