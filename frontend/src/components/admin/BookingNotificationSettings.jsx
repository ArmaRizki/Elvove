import { useEffect, useState } from "react";
import { LoaderCircle, Plus, Save, Trash2 } from "lucide-react";
import { adminApi } from "../../services/adminApi";
import {
  adminCard,
  adminInput,
  adminNotice,
  adminPrimaryButton,
  adminSecondaryButton,
} from "./adminStyles";

const EMAIL_PATTERN = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
const MAX_RECIPIENTS = 20;

function validate(rows) {
  const seen = new Set();
  for (const row of rows) {
    const email = row.value.trim();
    if (!email) return "Alamat email penerima tidak boleh kosong.";
    if (email.length > 255 || !EMAIL_PATTERN.test(email)) {
      return `Format email tidak valid: ${email}`;
    }
    const key = email.toLowerCase();
    if (seen.has(key)) return `Email penerima duplikat: ${email}`;
    seen.add(key);
  }
  return "";
}

export default function BookingNotificationSettings() {
  const [enabled, setEnabled] = useState(false);
  const [rows, setRows] = useState([]);
  const [nextId, setNextId] = useState(1);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [notice, setNotice] = useState({ type: "", text: "" });

  useEffect(() => {
    adminApi.getBookingNotificationSettings()
      .then(({ data }) => {
        setEnabled(Boolean(data.enabled));
        const list = Array.isArray(data.recipients) ? data.recipients : [];
        setRows(list.map((email, index) => ({ id: index + 1, value: email })));
        setNextId(list.length + 1);
      })
      .catch((e) => setNotice({
        type: "error",
        text: e.response?.data?.message || "Gagal memuat pengaturan notifikasi booking.",
      }))
      .finally(() => setLoading(false));
  }, []);

  const addRow = () => {
    if (rows.length >= MAX_RECIPIENTS) return;
    setRows((current) => [...current, { id: nextId, value: "" }]);
    setNextId((id) => id + 1);
  };

  const updateRow = (id, value) =>
    setRows((current) => current.map((row) => (row.id === id ? { ...row, value } : row)));

  const removeRow = (id) => setRows((current) => current.filter((row) => row.id !== id));

  const save = async () => {
    const problem = validate(rows);
    if (problem) return setNotice({ type: "error", text: problem });

    setSaving(true);
    setNotice({ type: "", text: "" });
    try {
      const recipients = rows.map((row) => row.value.trim());
      const { data } = await adminApi.saveBookingNotificationSettings({ enabled, recipients });
      setEnabled(Boolean(data.enabled));
      setRows(data.recipients.map((email, index) => ({ id: index + 1, value: email })));
      setNextId(data.recipients.length + 1);
      setNotice({ type: "success", text: "Pengaturan notifikasi booking berhasil disimpan." });
      if (data.enabled && data.recipients.length === 0) {
        setNotice({
          type: "warning",
          text: "Notifikasi aktif, tetapi belum ada penerima. Email tidak akan dikirim sampai penerima ditambahkan.",
        });
      }
    } catch (e) {
      setNotice({
        type: "error",
        text: e.response?.data?.message || "Gagal menyimpan pengaturan notifikasi booking.",
      });
    } finally {
      setSaving(false);
    }
  };

  if (loading) return null;
  const noticeClass = (type) =>
    type === "warning"
      ? "border border-amber-300 bg-amber-50 text-amber-900 p-4 text-sm"
      : adminNotice(type);

  return (
    <section className={`${adminCard} mt-6`} aria-labelledby="booking-notification-title">
      <h2 id="booking-notification-title" className="mb-2 font-display text-lg text-gray-900">
        Notifikasi Email Booking Baru
      </h2>
      <p className="mb-5 text-sm text-gray-600">
        Kirim email ke daftar penerima setiap kali pemesanan baru tersimpan. Email ini terpisah dari
        email konfirmasi untuk klien.
      </p>

      <label className="mb-5 flex items-center gap-2 text-sm text-gray-700" htmlFor="booking-notification-enabled">
        <input
          id="booking-notification-enabled"
          type="checkbox"
          checked={enabled}
          onChange={(e) => setEnabled(e.target.checked)}
        />
        Aktifkan Notifikasi Email Booking Baru
      </label>

      <div className="space-y-3">
        {rows.length === 0 ? (
          <p className="border border-dashed border-gray-300 p-4 text-sm text-gray-500">
            Belum ada email penerima.
          </p>
        ) : (
          rows.map((row, index) => (
            <div key={row.id} className="flex flex-col gap-2 sm:flex-row sm:items-center">
              <label className="sr-only" htmlFor={`booking-recipient-${row.id}`}>
                Email penerima {index + 1}
              </label>
              <input
                id={`booking-recipient-${row.id}`}
                type="email"
                className={`${adminInput} mt-0 min-w-0 sm:flex-1`}
                value={row.value}
                onChange={(e) => updateRow(row.id, e.target.value)}
                placeholder="admin@contoh.com"
                autoComplete="off"
              />
              <button
                type="button"
                onClick={() => removeRow(row.id)}
                className={`${adminSecondaryButton} w-full sm:w-auto`}
              >
                <Trash2 size={16} />
                Hapus
              </button>
            </div>
          ))
        )}
      </div>

      <button
        type="button"
        onClick={addRow}
        disabled={rows.length >= MAX_RECIPIENTS}
        className={`${adminSecondaryButton} mt-4 w-full sm:w-auto`}
      >
        <Plus size={16} />
        Tambah Email
      </button>

      {enabled && rows.length === 0 && (
        <p className="mt-4 border border-amber-300 bg-amber-50 p-3 text-sm text-amber-900">
          Notifikasi aktif, tetapi belum ada penerima. Email tidak akan dikirim.
        </p>
      )}

      {notice.text && (
        <div className={`mt-5 ${noticeClass(notice.type)}`} role="status">
          {notice.text}
        </div>
      )}

      <div className="mt-6 flex flex-col gap-2 sm:flex-row sm:flex-wrap">
        <button type="button" onClick={save} disabled={saving} className={`${adminPrimaryButton} w-full sm:w-auto`}>
          {saving ? <LoaderCircle size={16} className="animate-spin" /> : <Save size={16} />}
          Simpan Pengaturan
        </button>
      </div>
    </section>
  );
}
