import { useEffect, useState } from "react";
import { adminApi } from "../../services/adminApi";
import { Save, Eye, EyeOff, LoaderCircle } from "lucide-react";
import AdminPageHeader from "../../components/admin/AdminPageHeader";
import BookingNotificationSettings from "../../components/admin/BookingNotificationSettings";
import {
  adminCard,
  adminInput,
  adminLabel,
  adminNotice,
  adminPrimaryButton,
} from "../../components/admin/adminStyles";

const defaults = {
  host: "smtp.gmail.com", port: 587, username: "", password: "",
  encryption: "STARTTLS", authentication: true,
  senderName: "Elvove", senderEmail: "",
};

export default function SmtpSettings() {
  const [form, setForm] = useState(defaults);
  const [passwordConfigured, setPasswordConfigured] = useState(false);
  const [showPassword, setShowPassword] = useState(false);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [notice, setNotice] = useState({ type: "", text: "" });

  useEffect(() => {
    adminApi.getSmtpSettings()
      .then(({ data }) => {
        setForm((old) => ({ ...old, ...data, password: "" }));
        setPasswordConfigured(Boolean(data.passwordConfigured));
      })
      .catch((e) => setNotice({ type: "error", text: e.response?.data?.message || "Gagal memuat pengaturan SMTP." }))
      .finally(() => setLoading(false));
  }, []);

  const set = (key, value) => setForm((old) => ({ ...old, [key]: value }));

  const save = async (e) => {
    e.preventDefault(); setSaving(true); setNotice({ type: "", text: "" });
    try {
      const payload = { ...form, port: Number(form.port) };
      if (!payload.password) delete payload.password;
      const { data } = await adminApi.saveSmtpSettings(payload);
      setPasswordConfigured(Boolean(data.passwordConfigured));
      setForm((old) => ({ ...old, ...data, password: "" }));
      setNotice({ type: "success", text: "Pengaturan SMTP berhasil disimpan." });
    } catch (e) {
      setNotice({ type: "error", text: e.response?.data?.message || "Gagal menyimpan pengaturan SMTP." });
    } finally { setSaving(false); }
  };

  if (loading) return <p className="py-10 text-sm text-gray-500">Memuat pengaturan SMTP...</p>;

  return (
    <div className="min-w-0">
      <AdminPageHeader
        title="Pengaturan Email"
        description="Kelola server email untuk notifikasi dan konfirmasi pemesanan ELVOVE."
      />

      {notice.text && (
        <div className={`mb-6 ${adminNotice(notice.type)}`} role="status">
          {notice.text}
        </div>
      )}

      <form onSubmit={save} className="space-y-6">
        <section className={adminCard}>
          <h2 className="mb-4 font-display text-lg text-gray-900">Server SMTP</h2>

          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div className="sm:col-span-2">
              <label className={adminLabel} htmlFor="smtp-host">HOST SMTP</label>
              <input id="smtp-host" className={adminInput} required value={form.host}
                onChange={(e) => set("host", e.target.value)} placeholder="smtp.gmail.com" />
            </div>

            <div>
              <label className={adminLabel} htmlFor="smtp-port">PORT SMTP</label>
              <input id="smtp-port" className={adminInput} required type="number" min="1" max="65535"
                value={form.port} onChange={(e) => set("port", e.target.value)} />
            </div>

            <div>
              <label className={adminLabel} htmlFor="smtp-encryption">ENKRIPSI</label>
              <select id="smtp-encryption" className={adminInput} value={form.encryption}
                onChange={(e) => set("encryption", e.target.value)}>
                <option value="STARTTLS">STARTTLS</option>
                <option value="SSL">SSL / TLS</option>
                <option value="NONE">Tidak ada</option>
              </select>
            </div>

            <div className="sm:col-span-2">
              <label className={adminLabel} htmlFor="smtp-username">NAMA PENGGUNA / EMAIL</label>
              <input id="smtp-username" className={adminInput} value={form.username}
                onChange={(e) => set("username", e.target.value)} autoComplete="username" />
            </div>

            <div className="sm:col-span-2">
              <label className={adminLabel} htmlFor="smtp-password">KATA SANDI SMTP / KATA SANDI APLIKASI</label>
              <div className="relative">
                <input id="smtp-password" className={`${adminInput} pr-12`} type={showPassword ? "text" : "password"}
                  value={form.password} onChange={(e) => set("password", e.target.value)}
                  autoComplete="new-password"
                  placeholder={passwordConfigured ? "Kata sandi tersimpan. Kosongkan jika tidak diganti." : "Masukkan kata sandi SMTP"} />
                <button type="button" onClick={() => setShowPassword((v) => !v)}
                  className="absolute inset-y-0 right-0 px-3 text-gray-500" aria-label={showPassword ? "Sembunyikan kata sandi" : "Tampilkan kata sandi"}>
                  {showPassword ? <EyeOff size={17} /> : <Eye size={17} />}
                </button>
              </div>
              <p className="mt-1 text-xs text-gray-500">
                Kata sandi dienkripsi sebelum disimpan dan tidak pernah dikirim kembali ke browser.
              </p>
            </div>
          </div>

          <label className="mt-5 flex items-center gap-2 text-sm text-gray-700" htmlFor="smtp-auth">
            <input id="smtp-auth" type="checkbox" checked={Boolean(form.authentication)}
              onChange={(e) => set("authentication", e.target.checked)} />
            Autentikasi SMTP
          </label>
        </section>

        <section className={adminCard}>
          <h2 className="mb-4 font-display text-lg text-gray-900">Informasi Pengirim</h2>
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div>
              <label className={adminLabel} htmlFor="sender-name">NAMA PENGIRIM</label>
              <input id="sender-name" className={adminInput} required value={form.senderName}
                onChange={(e) => set("senderName", e.target.value)} />
            </div>
            <div>
              <label className={adminLabel} htmlFor="sender-email">EMAIL PENGIRIM</label>
              <input id="sender-email" className={adminInput} required type="email" value={form.senderEmail}
                onChange={(e) => set("senderEmail", e.target.value)} />
            </div>
          </div>
        </section>

        <div className="flex flex-col gap-2 sm:flex-row sm:flex-wrap">
          <button disabled={saving} className={`${adminPrimaryButton} w-full sm:w-auto`}>
            {saving ? <LoaderCircle size={16} className="animate-spin" /> : <Save size={16} />}
            Simpan Pengaturan
          </button>
        </div>
      </form>

      <BookingNotificationSettings />
    </div>
  );
}
