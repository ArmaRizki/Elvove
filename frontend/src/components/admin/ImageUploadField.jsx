import { useRef, useState } from "react";
import { LoaderCircle, Upload } from "lucide-react";
import { adminApi } from "../../services/adminApi";
import { adminLabel } from "./adminStyles";
import { getImageUrl } from "../../utils/imageUrl";
import {
  ACCEPTED_IMAGE_EXTENSIONS,
  MAX_FILE_LABEL,
  validatePhotoFile,
} from "../../utils/uploadLimits";

/**
 * Image picker shared by the photo and About editors.
 *
 * <p>Checks the file against the same limits the backend enforces, shows the
 * upload percentage, and ignores a second click while the first upload is still
 * running. Only the resulting URL is handed back to the parent form.</p>
 */
export default function ImageUploadField({
  id,
  label = "FOTO",
  value,
  onChange,
  onError,
  onUploadingChange,
  previewClassName = "h-14 w-14",
}) {
  const [uploading, setUploading] = useState(false);
  const [progress, setProgress] = useState(0);
  const [error, setError] = useState("");
  // Synchronous guard: state only updates after the next render.
  const inFlight = useRef(false);
  const inputRef = useRef(null);

  const fail = (message) => {
    setError(message);
    onError?.(message);
  };

  const upload = async (file) => {
    if (!file || inFlight.current) return;

    const problem = validatePhotoFile(file);
    if (problem) {
      fail(problem);
      return;
    }

    inFlight.current = true;
    setUploading(true);
    onUploadingChange?.(true);
    setProgress(0);
    setError("");
    try {
      const res = await adminApi.uploadFile(file, setProgress);
      onChange(res.data.url);
    } catch (err) {
      fail(
        err.response?.data?.message ||
          "Gagal mengunggah gambar. Periksa koneksi lalu coba lagi."
      );
    } finally {
      inFlight.current = false;
      setUploading(false);
      onUploadingChange?.(false);
      setProgress(0);
      // Allow re-picking the same file after a failure.
      if (inputRef.current) inputRef.current.value = "";
    }
  };

  return (
    <div>
      {/* A span, not a label: the clickable label wraps the file input below. */}
      <span className={adminLabel}>{label}</span>
      <div className="mt-1 flex flex-wrap items-center gap-3">
        {value && (
          <img
            src={getImageUrl(value)}
            alt=""
            className={`${previewClassName} border border-gray-200 object-cover`}
          />
        )}
        <label
          className={`flex cursor-pointer items-center gap-2 border border-gray-200 px-3 py-2 text-xs text-gray-700 hover:text-gray-900 ${
            uploading ? "cursor-not-allowed opacity-60" : ""
          }`}
        >
          {uploading ? (
            <LoaderCircle size={14} className="animate-spin" />
          ) : (
            <Upload size={14} />
          )}
          {uploading ? `Mengunggah... ${progress}%` : "Unggah Foto"}
          <input
            ref={inputRef}
            id={id}
            type="file"
            accept={ACCEPTED_IMAGE_EXTENSIONS}
            className="hidden"
            disabled={uploading}
            onChange={(e) => upload(e.target.files?.[0])}
          />
        </label>
      </div>

      {uploading && (
        <div
          className="mt-2 h-1 w-full max-w-xs overflow-hidden bg-gray-200"
          role="progressbar"
          aria-valuenow={progress}
          aria-valuemin={0}
          aria-valuemax={100}
        >
          <div className="h-full bg-[#E11D48] transition-all" style={{ width: `${progress}%` }} />
        </div>
      )}

      <p className="mt-2 text-xs text-gray-500">
        Format JPG, PNG, atau WebP. Maksimal {MAX_FILE_LABEL} per foto.
      </p>

      {error && (
        <p className="mt-2 text-xs text-red-700" role="status">
          {error}
        </p>
      )}
    </div>
  );
}
