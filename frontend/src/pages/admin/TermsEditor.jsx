import { useEffect, useState } from "react";
import { adminApi } from "../../services/adminApi";
import LoadingState from "../../components/LoadingState";
import AdminPageHeader from "../../components/admin/AdminPageHeader";
import RichTextEditor from "../../components/admin/RichTextEditor";
import { toHtml } from "../../utils/richText";
import {
  adminCard,
  adminInput,
  adminLabel,
  adminNotice,
  adminPrimaryButton,
} from "../../components/admin/adminStyles";

const COLUMN_COUNTS = [
  { value: 1, label: "Satu kolom (isi mengalir penuh)" },
  { value: 2, label: "Dua kolom" },
];

const COLUMN_WIDTH_OPTIONS = [
  { value: "equal", label: "Sama lebar" },
  { value: "wide-left", label: "Kolom kiri lebih lebar" },
  { value: "wide-right", label: "Kolom kanan lebih lebar" },
];

const SPLIT_OPTIONS = [
  { value: "heading", label: "Pada setiap Judul 1" },
  { value: "marker", label: "Pada penanda “Pisah kolom”" },
];

export default function TermsEditor() {
  const [form, setForm] = useState({
    title: "",
    content: "",
    version: "1",
  });
  const [layout, setLayout] = useState({
    layoutColumns: 2,
    layoutWidth: "equal",
    layoutSplit: "heading",
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
      // Older rows hold Markdown; the editor works on HTML.
      setForm({ ...data, content: toHtml(data.content) });
      setLayout({
        layoutColumns: data.layoutColumns === 1 ? 1 : 2,
        layoutWidth: data.layoutWidth || "equal",
        layoutSplit: data.layoutSplit || "heading",
      });
    } catch (error) {
      setNotice({
        type: "error",
        text: error.response?.data?.message || "Gagal memuat Syarat dan Ketentuan.",
      });
    } finally {
      setLoading(false);
    }
  }

  function setLayoutField(name, value) {
    setLayout((current) => ({ ...current, [name]: value }));
  }

  async function handleSave() {
    // The editor leaves "<p></p>" behind, so strip tags and &nbsp; first. The
    // page heading is fixed, so only the content is checked.
    if (!form.content.replace(/<[^>]*>|&nbsp;/g, "").trim()) {
      setNotice({ type: "error", text: "Isi wajib diisi." });
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
        ...layout,
      });
      setForm({ ...data, content: toHtml(data.content) });
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

  const columnsHint =
    layout.layoutColumns === 1
      ? "Seluruh isi tampil dalam satu kolom di halaman publik."
      : `Isi dibagi menjadi kolom baru ${
          layout.layoutSplit === "marker"
            ? "setiap kali Anda menekan tombol “Pisah kolom”."
            : "pada setiap Judul 1."
        }`;

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
          <RichTextEditor
            id="terms-content"
            label="ISI SYARAT DAN KETENTUAN"
            value={form.content}
            onChange={(content) => setForm((current) => ({ ...current, content }))}
            hint={`Gunakan Judul 2 dan Judul 3 untuk sub-bagian. ${columnsHint}`}
            minHeight="24rem"
            columnBreak={layout.layoutColumns === 2 && layout.layoutSplit === "marker"}
          />
        </div>

        <fieldset className="space-y-4 border-t border-gray-200 pt-5">
          <legend className="sr-only">Tata letak kolom</legend>

          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <label className={adminLabel} htmlFor="terms-columns">
              JUMLAH KOLOM
              <select
                id="terms-columns"
                value={layout.layoutColumns}
                onChange={(event) => setLayoutField("layoutColumns", Number(event.target.value))}
                className={adminInput}
              >
                {COLUMN_COUNTS.map(({ value, label }) => (
                  <option key={value} value={value}>{label}</option>
                ))}
              </select>
            </label>

            <label className={adminLabel} htmlFor="terms-split">
              KOLOM BARU DIMULAI
              <select
                id="terms-split"
                value={layout.layoutSplit}
                onChange={(event) => setLayoutField("layoutSplit", event.target.value)}
                disabled={layout.layoutColumns === 1}
                className={`${adminInput} disabled:opacity-50`}
              >
                {SPLIT_OPTIONS.map(({ value, label }) => (
                  <option key={value} value={value}>{label}</option>
                ))}
              </select>
            </label>

            <label className={adminLabel} htmlFor="terms-width">
              LEBAR KOLOM
              <select
                id="terms-width"
                value={layout.layoutWidth}
                onChange={(event) => setLayoutField("layoutWidth", event.target.value)}
                disabled={layout.layoutColumns === 1}
                className={`${adminInput} disabled:opacity-50`}
              >
                {COLUMN_WIDTH_OPTIONS.map(({ value, label }) => (
                  <option key={value} value={value}>{label}</option>
                ))}
              </select>
            </label>
          </div>

          <p className="text-xs text-gray-500">
            Di layar ponsel, kolom selalu ditumpuk menjadi satu.
          </p>
        </fieldset>

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
