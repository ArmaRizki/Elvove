import { useEffect, useState } from "react";
import { Plus, Pencil, Trash2, X } from "lucide-react";
import { adminApi } from "../../services/adminApi";
import LoadingState from "../../components/LoadingState";
import EmptyState from "../../components/EmptyState";
import AdminPageHeader from "../../components/admin/AdminPageHeader";
import {
  adminIconButton,
  adminInput,
  adminLabel,
  adminPrimaryButton,
  adminTable,
  adminTableWrap,
} from "../../components/admin/adminStyles";

// Packages gets a dedicated page (rather than the generic AdminResourceManager)
// because "features" is a list, entered here as one line per feature.
export default function Packages() {
  const [items, setItems] = useState([]);
  const [loading, setLoading] = useState(true);
  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState(null);
  const [form, setForm] = useState({ title: "", price: "", featuresText: "", displayOrder: 0, highlighted: false });
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");

  const load = () => {
    setLoading(true);
    adminApi.getPackages().then((res) => setItems(res.data)).finally(() => setLoading(false));
  };

  useEffect(() => {
    load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const openCreate = () => {
    setEditing(null);
    setForm({ title: "", price: "", featuresText: "", displayOrder: 0, highlighted: false });
    setError("");
    setModalOpen(true);
  };

  const openEdit = (item) => {
    setEditing(item);
    setForm({
      title: item.title,
      price: item.price,
      featuresText: (item.features || []).join("\n"),
      displayOrder: item.displayOrder,
      highlighted: item.highlighted,
    });
    setError("");
    setModalOpen(true);
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    setSaving(true);
    setError("");
    const payload = {
      title: form.title,
      price: Number(form.price),
      features: form.featuresText.split("\n").map((f) => f.trim()).filter(Boolean),
      displayOrder: Number(form.displayOrder) || 0,
      highlighted: form.highlighted,
    };
    try {
      if (editing) await adminApi.updatePackage(editing.id, payload);
      else await adminApi.createPackage(payload);
      setModalOpen(false);
      load();
    } catch (err) {
      setError(err.response?.data?.message || "Terjadi kesalahan. Periksa kembali formulir Anda.");
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async (item) => {
    if (!window.confirm(`Hapus paket "${item.title}"? Tindakan ini tidak dapat dibatalkan.`)) return;
    await adminApi.deletePackage(item.id);
    load();
  };

  const priceOf = (item) => `Rp${Number(item.price).toLocaleString("id-ID")}`;

  return (
    <div className="min-w-0">
      <AdminPageHeader
        title="Paket"
        description="Daftar paket harga yang ditampilkan pada halaman paket publik."
        actions={
          <button onClick={openCreate} className={adminPrimaryButton}>
            <Plus size={16} /> Tambah Baru
          </button>
        }
      />

      {loading ? (
        <LoadingState label="Memuat paket" />
      ) : items.length === 0 ? (
        <EmptyState message="Belum ada paket. Tambahkan paket pertama Anda." />
      ) : (
        <>
          <div className={`hidden sm:block ${adminTableWrap}`}>
            <table className={adminTable}>
              <thead>
                <tr className="border-b border-gray-200 text-left text-gray-700">
                  <th className="px-4 py-3 font-normal">Judul</th>
                  <th className="px-4 py-3 font-normal">Harga</th>
                  <th className="px-4 py-3 font-normal">Unggulan</th>
                  <th className="px-4 py-3" />
                </tr>
              </thead>
              <tbody>
                {items.map((item) => (
                  <tr key={item.id} className="border-b border-gray-200 last:border-0">
                    <td className="px-4 py-3 text-gray-700">{item.title}</td>
                    <td className="px-4 py-3 text-gray-700">{priceOf(item)}</td>
                    <td className="px-4 py-3 text-gray-700">{item.highlighted ? "Ya" : "—"}</td>
                    <td className="px-4 py-3 text-right whitespace-nowrap">
                      <button
                        onClick={() => openEdit(item)}
                        className={`${adminIconButton} hover:text-[#E11D48]`}
                        aria-label={`Ubah paket ${item.title}`}
                      >
                        <Pencil size={15} />
                      </button>
                      <button
                        onClick={() => handleDelete(item)}
                        className={`${adminIconButton} hover:text-burgundy`}
                        aria-label={`Hapus paket ${item.title}`}
                      >
                        <Trash2 size={15} />
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          <div className="space-y-3 sm:hidden">
            {items.map((item) => (
              <article key={item.id} className="flex items-start justify-between gap-3 border border-gray-200 p-4">
                <div className="min-w-0">
                  <p className="break-words font-medium text-gray-900">{item.title}</p>
                  <p className="text-sm text-gray-700">{priceOf(item)}</p>
                  {item.highlighted && (
                    <p className="mt-1 text-xs text-[#E11D48]">Paling Banyak Dipesan</p>
                  )}
                </div>
                <div className="flex shrink-0 gap-1">
                  <button
                    onClick={() => openEdit(item)}
                    className={adminIconButton}
                    aria-label={`Ubah paket ${item.title}`}
                  >
                    <Pencil size={16} />
                  </button>
                  <button
                    onClick={() => handleDelete(item)}
                    className={adminIconButton}
                    aria-label={`Hapus paket ${item.title}`}
                  >
                    <Trash2 size={16} />
                  </button>
                </div>
              </article>
            ))}
          </div>
        </>
      )}

      {modalOpen && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-white/80 p-4 sm:p-6"
          onClick={() => setModalOpen(false)}
        >
          <div
            role="dialog"
            aria-modal="true"
            aria-label={`${editing ? "Ubah" : "Tambah"} Paket`}
            className="max-h-[90vh] w-full max-w-lg overflow-y-auto border border-gray-200 bg-white"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="flex items-center justify-between border-b border-gray-200 px-4 py-4 sm:px-6">
              <h2 className="font-display text-lg text-gray-900">{editing ? "Ubah" : "Tambah"} Paket</h2>
              <button
                onClick={() => setModalOpen(false)}
                className="p-2 text-gray-700 hover:text-gray-900"
                aria-label="Tutup"
              >
                <X size={20} />
              </button>
            </div>
            <form onSubmit={handleSubmit} className="space-y-4 p-4 sm:p-6">
              <div>
                <label className={adminLabel} htmlFor="package-title">JUDUL</label>
                <input
                  id="package-title"
                  value={form.title}
                  onChange={(e) => setForm({ ...form, title: e.target.value })}
                  required
                  className={adminInput}
                />
              </div>
              <div>
                <label className={adminLabel} htmlFor="package-price">HARGA (IDR)</label>
                <input
                  id="package-price"
                  type="number"
                  value={form.price}
                  onChange={(e) => setForm({ ...form, price: e.target.value })}
                  required
                  className={adminInput}
                />
              </div>
              <div>
                <label className={adminLabel} htmlFor="package-features">FITUR (SATU PER BARIS)</label>
                <textarea
                  id="package-features"
                  rows={5}
                  value={form.featuresText}
                  onChange={(e) => setForm({ ...form, featuresText: e.target.value })}
                  className={adminInput}
                />
              </div>
              <div>
                <label className={adminLabel} htmlFor="package-order">URUTAN TAMPIL</label>
                <input
                  id="package-order"
                  type="number"
                  value={form.displayOrder}
                  onChange={(e) => setForm({ ...form, displayOrder: e.target.value })}
                  className={adminInput}
                />
              </div>
              <label className="flex items-center gap-2 text-sm text-gray-700" htmlFor="package-highlighted">
                <input
                  id="package-highlighted"
                  type="checkbox"
                  checked={form.highlighted}
                  onChange={(e) => setForm({ ...form, highlighted: e.target.checked })}
                  className="accent-brass"
                />
                Tandai sebagai &quot;Paling Banyak Dipesan&quot;
              </label>
              {error && <p className="text-burgundy text-sm">{error}</p>}
              <button type="submit" disabled={saving} className={`${adminPrimaryButton} w-full`}>
                {saving ? "Menyimpan..." : "Simpan"}
              </button>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
