// Aturan waktu sesi foto: hanya menit 00, 15, 30, dan 45.
export const MINUTE_STEP = 15;

export const pad = (value) => String(value).padStart(2, "0");

/** Tanggal hari ini menurut waktu lokal, dalam format YYYY-MM-DD. */
export const todayIso = () => {
  const now = new Date();
  return `${now.getFullYear()}-${pad(now.getMonth() + 1)}-${pad(now.getDate())}`;
};

/** Menit dari nilai "HH:mm", atau null bila bentuknya tidak dikenal. */
export const minutesOf = (value) => {
  const match = /^(\d{2}):(\d{2})/.exec(value || "");
  return match ? Number(match[2]) : null;
};

/**
 * Tanggal ISO "YYYY-MM-DD" menjadi "dd/mm/yyyy" untuk ditampilkan.
 * Nilai yang dikirim ke API tetap format ISO.
 */
export const formatDateDisplay = (isoDate) => {
  const match = /^(\d{4})-(\d{2})-(\d{2})$/.exec(isoDate || "");
  return match ? `${match[3]}/${match[2]}/${match[1]}` : "";
};

/**
 * True hanya untuk waktu pada kelipatan 15 menit. Dipakai sebagai penjaga
 * terakhir saat submit, terlepas dari nilai apa pun yang sampai ke form.
 */
export const isQuarterHour = (value) => {
  const minutes = minutesOf(value);
  return minutes !== null && minutes % MINUTE_STEP === 0;
};
