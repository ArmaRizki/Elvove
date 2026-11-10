import { useEffect, useState } from "react";
import { Plus, Pencil, Trash2, X, Upload } from "lucide-react";
import { adminApi } from "../services/adminApi";
import LoadingState from "./LoadingState";
import EmptyState from "./EmptyState";
import AdminPageHeader from "./admin/AdminPageHeader";
import {
  adminIconButton,
  adminInput,
  adminLabel,
  adminPrimaryButton,
  adminTable,
  adminTableWrap,
} from "./admin/adminStyles";
import { getImageUrl } from "../utils/imageUrl";
import {
  ACCEPTED_IMAGE_EXTENSIONS,
  MAX_FILE_LABEL,
  validatePhotoFile,
} from "../utils/uploadLimits";

/**
 * Generic CRUD manager used by every simple admin resource (Packages,
 * Categories, and the singleton editors). Photos gets its own page since it
 * also needs category selection wired to a live list, but follows this same shape.
 *
 * fields: [{ name, label, type: 'text'|'textarea'|'number'|'checkbox'|'image', required }]
 */
export default function AdminResourceManager({ title, fields, columns, api, emptyDefaults }) {
  const [items, setItems] = useState([]);
  const [loading, setLoading] = useState(true);
  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState(null);
  const [form, setForm] = useState(emptyDefaults);
  const [saving, setSaving] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [error, setError] = useState("");

  const load = () => {
    setLoading(true);
    api.list().then((res) => setItems(res.data)).finally(() => setLoading(false));
  };

  useEffect(() => {
    load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const openCreate = () => {
    setEditing(null);
    setForm(emptyDefaults);
    setError("");
    setModalOpen(true);
  };

  const openEdit = (item) => {
    setEditing(item);
    setForm({ ...emptyDefaults, ...item });
    setError("");
    setModalOpen(true);
  };

  const handleFileUpload = async (fieldName, file) => {
    if (!file) return;

    const problem = validatePhotoFile(file);
    if (problem) {
      setError(problem);
      return;
    }

    setUploading(true);
    setError("");
    try {
      const res = await adminApi.uploadFile(file);
      setForm((f) => ({ ...f, [fieldName]: res.data.url }));
    } catch (err) {
      setError(
        err.response?.data?.message || "Gagal mengunggah gambar. Silakan coba berkas lain."
      );
    } finally {
      setUploading(false);
    }
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    setSaving(true);
    setError("");
    try {
      if (editing) {
        await api.update(editing.id, form);
      } else {
        await api.create(form);
      }
      setModalOpen(false);
      load();
    } catch (err) {
      setError(err.response?.data?.message || "Terjadi kesalahan. Periksa kembali formulir Anda.");
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = async (item) => {
    if (!window.confirm(`Hapus "${item[columns[0].key]}"? Tindakan ini tidak dapat dibatalkan.`)) return;
    await api.delete(item.id);
    load();
  };

  return (
    <div className="min-w-0">
      <AdminPageHeader
        title={title}
        actions={
          <button onClick={openCreate} className={adminPrimaryButton}>
            <Plus size={16} /> Tambah Baru
          </button>
        }
      />

      {loading ? (
        <LoadingState label="Memuat data" />
      ) : items.length === 0 ? (
        <EmptyState message={`Belum ada data ${title.toLowerCase()}. Tambahkan data pertama Anda.`} />
      ) : (
        <>
          <div className={`hidden sm:block ${adminTableWrap}`}>
            <table className={adminTable}>
              <thead>
                <tr className="border-b border-gray-200 text-left text-gray-700">
                  {columns.map((c) => (
                    <th key={c.key} className="px-4 py-3 font-normal">{c.label}</th>
                  ))}
                  <th className="px-4 py-3" />
                </tr>
              </thead>
              <tbody>
                {items.map((item) => (
                  <tr key={item.id} className="border-b border-gray-200 last:border-0">
                    {columns.map((c) => (
                      <td key={c.key} className="px-4 py-3 text-gray-700">
                        {c.render ? c.render(item[c.key], item) : String(item[c.key] ?? "")}
                      </td>
                    ))}
                    <td className="px-4 py-3 text-right whitespace-nowrap">
                      <button
                        onClick={() => openEdit(item)}
                        className={`${adminIconButton} hover:text-[#E11D48]`}
                        aria-label={`Ubah ${item[columns[0].key]}`}
                      >
                        <Pencil size={15} />
                      </button>
                      <button
                        onClick={() => handleDelete(item)}
                        className={`${adminIconButton} hover:text-burgundy`}
                        aria-label={`Hapus ${item[columns[0].key]}`}
                      >
                        <Trash2 size={15} />
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          {/* Phones: stacked cards keep every column readable without scrolling */}
          <div className="space-y-3 sm:hidden">
            {items.map((item) => (
              <article key={item.id} className="border border-gray-200 p-4">
                <div className="flex items-start justify-between gap-3">
                  <div className="min-w-0 space-y-1">
                    {columns.map((c) => (
                      <p key={c.key} className="min-w-0 break-words text-sm text-gray-700">
                        <span className="mr-2 text-xs text-gray-500">{c.label}</span>
                        {c.render ? c.render(item[c.key], item) : String(item[c.key] ?? "")}
                      </p>
                    ))}
                  </div>
                  <div className="flex shrink-0 gap-1">
                    <button
                      onClick={() => openEdit(item)}
                      className={adminIconButton}
                      aria-label={`Ubah ${item[columns[0].key]}`}
                    >
                      <Pencil size={16} />
                    </button>
                    <button
                      onClick={() => handleDelete(item)}
                      className={adminIconButton}
                      aria-label={`Hapus ${item[columns[0].key]}`}
                    >
                      <Trash2 size={16} />
                    </button>
                  </div>
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
            aria-label={`${editing ? "Ubah" : "Tambah"} ${title}`}
            className="max-h-[90vh] w-full max-w-lg overflow-y-auto border border-gray-200 bg-white"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="flex items-center justify-between border-b border-gray-200 px-4 py-4 sm:px-6">
              <h2 className="font-display text-lg text-gray-900">
                {editing ? "Ubah" : "Tambah"} {title}
              </h2>
              <button
                onClick={() => setModalOpen(false)}
                className="p-2 text-gray-700 hover:text-gray-900"
                aria-label="Tutup"
              >
                <X size={20} />
              </button>
            </div>

            <form onSubmit={handleSubmit} className="space-y-4 p-4 sm:p-6">
              {fields.map((f) => (
                <div key={f.name}>
                  <label className={adminLabel} htmlFor={`field-${f.name}`}>
                    {f.label.toUpperCase()}
                  </label>

                  {f.type === "textarea" && (
                    <textarea
                      id={`field-${f.name}`}
                      rows={3}
                      value={form[f.name] ?? ""}
                      onChange={(e) => setForm({ ...form, [f.name]: e.target.value })}
                      className={adminInput}
                      required={f.required}
                    />
                  )}

                  {f.type === "checkbox" && (
                    <div className="mt-2">
                      <input
                        id={`field-${f.name}`}
                        type="checkbox"
                        checked={!!form[f.name]}
                        onChange={(e) => setForm({ ...form, [f.name]: e.target.checked })}
                        className="accent-brass"
                      />
                    </div>
                  )}

                  {f.type === "image" && (
                    <div className="mt-1 flex items-center gap-3">
                      {form[f.name] && (
                        <img src={getImageUrl(form[f.name])} alt="" className="h-14 w-14 border border-gray-200 object-cover" />
                      )}
                      <label className="flex cursor-pointer items-center gap-2 border border-gray-200 px-3 py-2 text-xs text-gray-700 hover:text-gray-900">
                        <Upload size={14} />
                        {uploading ? "Mengunggah..." : "Unggah gambar"}
                        <input
                          type="file"
                          accept={ACCEPTED_IMAGE_EXTENSIONS}
                          className="hidden"
                          onChange={(e) => handleFileUpload(f.name, e.target.files[0])}
                        />
                      </label>
                    </div>
                  )}

                  {f.type === "image" && (
                    <p className="mt-2 text-xs text-gray-500">
                      Format JPG, PNG, atau WebP. Maksimal {MAX_FILE_LABEL} per foto.
                    </p>
                  )}

                  {(f.type === "text" || f.type === "number" || f.type === undefined) && (
                    <input
                      id={`field-${f.name}`}
                      type={f.type === "number" ? "number" : "text"}
                      value={form[f.name] ?? ""}
                      onChange={(e) => setForm({ ...form, [f.name]: e.target.value })}
                      className={adminInput}
                      required={f.required}
                    />
                  )}
                </div>
              ))}

              {error && <p className="text-burgundy text-sm">{error}</p>}

              <button
                type="submit"
                disabled={saving || uploading}
                className={`${adminPrimaryButton} w-full`}
              >
                {saving ? "Menyimpan..." : "Simpan"}
              </button>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
