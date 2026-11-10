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

export default function ContactEditor() {
  const [form, setForm] = useState(null);
  const [saving, setSaving] = useState(false);
  const [notice, setNotice] = useState({ type: "", text: "" });

  useEffect(() => {
    adminApi
      .getContact()
      .then((res) => setForm(res.data))
      .catch(() => setNotice({ type: "error", text: "Gagal memuat informasi kontak." }));
  }, []);

  const handleSubmit = async (e) => {
    e.preventDefault();
    setSaving(true);
    setNotice({ type: "", text: "" });
    try {
      await adminApi.updateContact(form);
      setNotice({ type: "success", text: "Informasi kontak berhasil disimpan." });
    } catch (err) {
      setNotice({
        type: "error",
        text: err.response?.data?.message || "Gagal menyimpan informasi kontak.",
      });
    } finally {
      setSaving(false);
    }
  };

  if (!form) {
    return notice.text ? (
      <p className="border border-red-200 bg-red-50 p-3 text-sm text-red-700">{notice.text}</p>
    ) : (
      <LoadingState label="Memuat informasi kontak" />
    );
  }

  return (
    <div className="min-w-0">
      <AdminPageHeader
        title="Informasi Kontak"
        description="Detail kontak yang ditampilkan pada footer dan halaman kontak."
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
          <div>
            <label className={adminLabel} htmlFor="contact-phone">TELEPON</label>
            <input
              id="contact-phone"
              value={form.phone || ""}
              onChange={(e) => setForm({ ...form, phone: e.target.value })}
              className={adminInput}
            />
          </div>
          <div>
            <label className={adminLabel} htmlFor="contact-email">EMAIL</label>
            <input
              id="contact-email"
              type="email"
              value={form.email || ""}
              onChange={(e) => setForm({ ...form, email: e.target.value })}
              className={adminInput}
            />
          </div>
        </div>

        <div>
          <label className={adminLabel} htmlFor="contact-address">ALAMAT</label>
          <textarea
            id="contact-address"
            rows={2}
            value={form.address || ""}
            onChange={(e) => setForm({ ...form, address: e.target.value })}
            className={adminInput}
          />
        </div>

        <div>
          <label className={adminLabel} htmlFor="contact-map">TAUTAN SEMATAN GOOGLE MAPS</label>
          <input
            id="contact-map"
            value={form.googleMapLink || ""}
            onChange={(e) => setForm({ ...form, googleMapLink: e.target.value })}
            className={adminInput}
          />
        </div>

        <div>
          <label className={adminLabel} htmlFor="contact-whatsapp">
            NOMOR WHATSAPP (DENGAN KODE NEGARA)
          </label>
          <input
            id="contact-whatsapp"
            value={form.whatsappNumber || ""}
            onChange={(e) => setForm({ ...form, whatsappNumber: e.target.value })}
            className={adminInput}
          />
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
