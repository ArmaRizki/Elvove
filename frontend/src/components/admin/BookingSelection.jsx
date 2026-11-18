import { useState } from "react";
import {
  Check,
  Copy,
  ExternalLink,
  FolderOpen,
  FolderPlus,
  LoaderCircle,
  RefreshCw,
} from "lucide-react";
import { adminApi } from "../../services/adminApi";
import { adminInput, adminLabel, adminNotice, adminSecondaryButton } from "./adminStyles";
import { formatDateTime } from "./adminFormat";

/**
 * The raw photo folder, the client's selection link and how far the client got.
 *
 * <p>The folder is created automatically when the booking comes in, with its
 * webp, "foto terpilih" and "foto edit" subfolders, so nothing has to be pasted
 * here. The button is the retry for a booking taken while Google Drive was not
 * connected, and it shares the "foto edit" folder as viewer.</p>
 */
export default function BookingSelection({ booking, onUpdated }) {
  const [busy, setBusy] = useState(false);
  const [notice, setNotice] = useState({ type: "", text: "" });
  const [copied, setCopied] = useState(false);

  if (!booking) return null;

  const folderId = (booking.rawFolderId || "").trim();
  const limit = Number(booking.selectionLimit) || 0;
  const photoCount = Number(booking.photoCount) || 0;
  const selectedCount = Number(booking.selectedCount) || 0;
  const submittedAt = formatDateTime(booking.selectionSubmittedAt);
  const clientURL = booking.selectionToken
    ? `${window.location.origin}/pilih/${booking.selectionToken}`
    : "";

  const createFolder = async () => {
    setBusy(true);
    setNotice({ type: "", text: "" });
    try {
      const { data } = await adminApi.createBookingDriveFolder(booking.id);
      onUpdated?.(data);
      setNotice({
        type: "success",
        text: 'Folder foto klien siap di Google Drive: subfolder webp, "foto terpilih" dan "foto edit" yang dibagikan sebagai pelihat.',
      });
    } catch (error) {
      setNotice({
        type: "error",
        text: error.response?.data?.message || "Folder foto gagal dibuat.",
      });
    } finally {
      setBusy(false);
    }
  };

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(clientURL);
      setCopied(true);
      window.setTimeout(() => setCopied(false), 2000);
    } catch {
      setNotice({
        type: "error",
        text: "Link tidak bisa disalin otomatis. Salin manual dari kolom di bawah.",
      });
    }
  };

  return (
    <div className="space-y-4">
      <div>
        <p className={adminLabel}>FOLDER FOTO KLIEN (GOOGLE DRIVE)</p>

        {folderId ? (
          <div className="mt-1 flex flex-wrap items-center gap-3 border border-green-200 bg-green-50 p-3 text-sm text-green-900">
            <FolderOpen size={16} className="shrink-0" />
            <span className="min-w-0 flex-1 truncate font-mono text-xs">{folderId}</span>
            <a
              href={driveFolderLink(folderId)}
              target="_blank"
              rel="noreferrer"
              className="inline-flex items-center gap-1 text-xs underline"
            >
              Buka <ExternalLink size={12} />
            </a>
            <button
              type="button"
              onClick={createFolder}
              disabled={busy}
              className={`${adminSecondaryButton} py-1.5`}
            >
              {busy ? (
                <LoaderCircle size={14} className="animate-spin" />
              ) : (
                <RefreshCw size={14} />
              )}
              Periksa
            </button>
          </div>
        ) : (
          <div className="mt-1 flex flex-wrap items-center gap-3 border border-amber-200 bg-amber-50 p-3 text-sm text-amber-900">
            <span className="min-w-0 flex-1">
              Folder belum dibuat. Sambungkan Google Drive di menu Google Drive, lalu buat di sini.
            </span>
            <button
              type="button"
              onClick={createFolder}
              disabled={busy}
              className={`${adminSecondaryButton} shrink-0 py-1.5`}
            >
              {busy ? <LoaderCircle size={14} className="animate-spin" /> : <FolderPlus size={14} />}
              Buat Folder
            </button>
          </div>
        )}

        <p className="mt-1 text-xs text-gray-500">
          Elvove membuat folder ini otomatis setiap ada pemesanan. Unggah foto mentah ke dalamnya;
          pratinjau WebP masuk ke subfolder
          <span className="mx-1 font-mono">webp</span>, foto pilihan klien disalin ke
          <span className="mx-1 font-mono">foto terpilih</span>, dan hasil edit Anda taruh di
          <span className="mx-1 font-mono">foto edit</span> (link-nya otomatis masuk invoice).
          Folder ini dibagikan sebagai pelihat, jadi tombol Buka bisa dibuka dari akun Google mana
          pun tanpa minta akses.
        </p>
      </div>

      {clientURL && (
        <div>
          <label className={adminLabel} htmlFor="booking-selection-link">
            LINK PILIHAN FOTO UNTUK KLIEN
          </label>
          <div className="mt-1 flex flex-col gap-2 sm:flex-row">
            <input id="booking-selection-link" readOnly value={clientURL} className={adminInput} />
            <div className="flex shrink-0 gap-2">
              <button type="button" onClick={copy} className={adminSecondaryButton}>
                {copied ? <Check size={16} /> : <Copy size={16} />}
                {copied ? "Tersalin" : "Salin"}
              </button>
              <a
                href={clientURL}
                target="_blank"
                rel="noreferrer"
                className={adminSecondaryButton}
                aria-label="Buka halaman pilihan foto"
              >
                <ExternalLink size={16} />
              </a>
            </div>
          </div>
          <p className="mt-1 text-xs text-gray-500">
            Link ini dikirim lewat tombol <strong>Kirim Email Pilihan Foto</strong> di bawah, sebelum
            email pelunasan.
          </p>
        </div>
      )}

      <div className="grid grid-cols-1 gap-2 text-xs text-gray-600 sm:grid-cols-2">
        <p className="flex items-center gap-2">
          <FolderOpen size={14} className="shrink-0 text-gray-400" />
          {limit > 0 ? `Kuota pilih ${limit} foto` : "Paket ini belum punya kuota pilih foto"}
        </p>
        <p>
          {photoCount} foto terdeteksi · {selectedCount} dipilih klien
        </p>
        {submittedAt && <p className="text-green-700">Pilihan dikirim pada {submittedAt}</p>}
      </div>

      {notice.text && (
        <p className={adminNotice(notice.type)} role="status">
          {notice.text}
        </p>
      )}
    </div>
  );
}

// driveFolderLink turns the stored folder id back into a link an admin can open.
function driveFolderLink(id) {
  return `https://drive.google.com/drive/folders/${id}`;
}
