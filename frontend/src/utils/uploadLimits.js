// Upload limits shown to the admin and checked before a request is sent.
// These mirror the backend defaults in application.yml
// (UPLOAD_MAX_FILE_SIZE / UPLOAD_MAX_REQUEST_SIZE). The backend check is the
// authoritative one; this only saves the admin a failed upload.

const DEFAULT_MAX_FILE_MB = 50;
const DEFAULT_MAX_REQUEST_MB = 100;

const configuredFileMb = Number(import.meta.env.VITE_UPLOAD_MAX_FILE_SIZE_MB);
const configuredRequestMb = Number(import.meta.env.VITE_UPLOAD_MAX_REQUEST_SIZE_MB);

export const MAX_FILE_MB =
  Number.isFinite(configuredFileMb) && configuredFileMb > 0
    ? configuredFileMb
    : DEFAULT_MAX_FILE_MB;

export const MAX_REQUEST_MB =
  Number.isFinite(configuredRequestMb) && configuredRequestMb > 0
    ? configuredRequestMb
    : DEFAULT_MAX_REQUEST_MB;

export const MAX_FILE_BYTES = MAX_FILE_MB * 1024 * 1024;

// Same formats FileStorageService accepts: JPEG, PNG and WebP.
export const ALLOWED_IMAGE_TYPES = ["image/jpeg", "image/png", "image/webp"];

const ALLOWED_EXTENSIONS = /\.(jpe?g|png|webp)$/i;

/** Value for a file input's accept attribute. */
export const ACCEPTED_IMAGE_EXTENSIONS = ".jpg,.jpeg,.png,.webp";

/** "50 MB" — used in the size hints and the error messages. */
export const MAX_FILE_LABEL = `${MAX_FILE_MB} MB`;

/**
 * Returns an Indonesian error message for a file that cannot be uploaded, or
 * null when the file is acceptable.
 */
export function validatePhotoFile(file) {
  if (!file) return "Pilih berkas foto terlebih dahulu.";

  const name = file.name || "";
  const typeOk = file.type ? ALLOWED_IMAGE_TYPES.includes(file.type) : false;
  if (!typeOk && !ALLOWED_EXTENSIONS.test(name)) {
    return "Format berkas tidak didukung. Gunakan foto JPG, PNG, atau WebP.";
  }

  if (file.size > MAX_FILE_BYTES) {
    const actualMb = (file.size / (1024 * 1024)).toFixed(1);
    return `Ukuran foto ${actualMb} MB melebihi batas maksimal ${MAX_FILE_LABEL}.`;
  }

  return null;
}
