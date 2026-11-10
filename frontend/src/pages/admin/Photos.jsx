import { useEffect, useState } from "react";
import { Plus, Pencil, Trash2, X } from "lucide-react";
import { adminApi } from "../../services/adminApi";
import LoadingState from "../../components/LoadingState";
import EmptyState from "../../components/EmptyState";
import AdminPageHeader from "../../components/admin/AdminPageHeader";
import ImageUploadField from "../../components/admin/ImageUploadField";
import {
  adminInput,
  adminLabel,
  adminPrimaryButton,
} from "../../components/admin/adminStyles";
import { getImageUrl } from "../../utils/imageUrl";

const emptyForm = {
  title: "",
  description: "",
  imageUrl: "",
  categoryId: "",
  featured: false,
  displayOrder: 0,
};

export default function Photos() {
  const [photos, setPhotos] = useState([]);
  const [categories, setCategories] = useState([]);
  const [loading, setLoading] = useState(true);
  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState(null);
  const [uploading, setUploading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [form, setForm] = useState(emptyForm);

  const load = () => {
    setLoading(true);
    Promise.all([adminApi.getPhotos(), adminApi.getCategories()])
      .then(([p, c]) => {
        setPhotos(p.data);
        setCategories(c.data);
      })
      .finally(() => setLoading(false));
  };

  useEffect(() => {
    load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const openCreate = () => {
    setEditing(null);
    setForm(emptyForm);
    setError("");
    setModalOpen(true);
  };

  const openEdit = (photo) => {
    setEditing(photo);
    setForm({ ...photo, categoryId: photo.categoryId });
    setError("");
    setModalOpen(true);
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    if (!form.categoryId) {
      setError("Pilih kategori terlebih dahulu.");
      return;
    }
    setSaving(true);
    setError("");
    const payload = { ...form, categoryId: Number(form.categoryId), displayOrder: Number(form.displayOrder) || 0 };
    try {
      if (editing) await adminApi.updatePhoto(editing.id, payload);
      else await adminApi.createPhoto(payload);
      setModalOpen(false);
      load();
    } catch (err) {
      setError(err.response?.data?.message || "Terjadi kesalahan. Silakan coba lagi.");
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async (photo) => {
    if (!window.confirm(`Hapus foto "${photo.title}"? Tindakan ini tidak dapat dibatalkan.`)) return;
    await adminApi.deletePhoto(photo.id);
    load();
  };

  return (
    <div className="min-w-0">
      <AdminPageHeader
        title="Foto"
        description="Unggah dan kelola foto galeri."
        actions={
          <button onClick={openCreate} className={adminPrimaryButton}>
            <Plus size={16} /> Tambah Baru
          </button>
        }
      />

      {loading ? (
        <LoadingState label="Memuat foto" />
      ) : photos.length === 0 ? (
        <EmptyState message="Belum ada foto. Tambahkan foto pertama Anda." />
      ) : (
        <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 sm:gap-4 lg:grid-cols-4">
          {photos.map((p) => (
            <div key={p.id} className="group relative border border-gray-200">
              <img
                src={getImageUrl(p.thumbnailUrl || p.imageUrl)}
                alt={p.title}
                loading="lazy"
                className="h-32 w-full object-cover sm:h-40"
              />
              <div className="p-3">
                <p className="truncate text-sm text-gray-900">{p.title}</p>
                <p className="truncate text-xs text-gray-700">{p.categoryName}</p>
              </div>
              <div className="absolute right-2 top-2 flex gap-1 opacity-100 transition-opacity sm:opacity-0 sm:group-hover:opacity-100 sm:group-focus-within:opacity-100">
                <button
                  onClick={() => openEdit(p)}
                  className="bg-white/90 p-1.5 text-gray-900 hover:text-[#E11D48]"
                  aria-label={`Ubah foto ${p.title}`}
                >
                  <Pencil size={14} />
                </button>
                <button
                  onClick={() => handleDelete(p)}
                  className="bg-white/90 p-1.5 text-gray-900 hover:text-burgundy"
                  aria-label={`Hapus foto ${p.title}`}
                >
                  <Trash2 size={14} />
                </button>
              </div>
            </div>
          ))}
        </div>
      )}

      {modalOpen && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-white/80 p-4 sm:p-6"
          onClick={() => setModalOpen(false)}
        >
          <div
            role="dialog"
            aria-modal="true"
            aria-label={`${editing ? "Ubah" : "Tambah"} Foto`}
            className="max-h-[90vh] w-full max-w-lg overflow-y-auto border border-gray-200 bg-white"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="flex items-center justify-between border-b border-gray-200 px-4 py-4 sm:px-6">
              <h2 className="font-display text-lg text-gray-900">{editing ? "Ubah" : "Tambah"} Foto</h2>
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
                <label className={adminLabel} htmlFor="photo-title">JUDUL</label>
                <input
                  id="photo-title"
                  value={form.title}
                  onChange={(e) => setForm({ ...form, title: e.target.value })}
                  required
                  className={adminInput}
                />
              </div>
              <div>
                <label className={adminLabel} htmlFor="photo-description">DESKRIPSI</label>
                <textarea
                  id="photo-description"
                  rows={2}
                  value={form.description}
                  onChange={(e) => setForm({ ...form, description: e.target.value })}
                  className={adminInput}
                />
              </div>
              <div>
                <label className={adminLabel} htmlFor="photo-category">KATEGORI</label>
                <select
                  id="photo-category"
                  value={form.categoryId}
                  onChange={(e) => setForm({ ...form, categoryId: e.target.value })}
                  className={adminInput}
                >
                  <option value="">Pilih kategori</option>
                  {categories.map((c) => <option key={c.id} value={c.id}>{c.name}</option>)}
                </select>
              </div>
              <div>
                <ImageUploadField
                  id="photo-image"
                  label="GAMBAR"
                  value={form.imageUrl}
                  onChange={(url) => setForm((f) => ({ ...f, imageUrl: url }))}
                  onError={setError}
                  onUploadingChange={setUploading}
                />
              </div>
              <label className="flex items-center gap-2 text-sm text-gray-700" htmlFor="photo-featured">
                <input
                  id="photo-featured"
                  type="checkbox"
                  checked={form.featured}
                  onChange={(e) => setForm({ ...form, featured: e.target.checked })}
                  className="accent-brass"
                />
                Tampilkan di beranda
              </label>
              <div>
                <label className={adminLabel} htmlFor="photo-order">URUTAN TAMPIL</label>
                <input
                  id="photo-order"
                  type="number"
                  value={form.displayOrder}
                  onChange={(e) => setForm({ ...form, displayOrder: e.target.value })}
                  className={adminInput}
                />
              </div>
              {error && <p className="text-burgundy text-sm">{error}</p>}
              <button type="submit" disabled={saving || uploading} className={`${adminPrimaryButton} w-full`}>
                {saving ? "Menyimpan..." : "Simpan"}
              </button>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
