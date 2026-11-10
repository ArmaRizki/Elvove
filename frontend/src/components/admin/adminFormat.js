// Indonesian labels and formatters shared by the admin panel, so statuses,
// dates and prices read the same way everywhere.

export const STATUS_LABELS = {
  PENDING: "Menunggu",
  ACCEPTED: "Diterima",
  RESCHEDULE_REQUESTED: "Minta Ubah Jadwal",
  COMPLETED: "Selesai",
};

const STATUS_BADGES = {
  PENDING: "border-rose-200 bg-rose-50 text-rose-700",
  ACCEPTED: "border-green-200 bg-green-50 text-green-800",
  RESCHEDULE_REQUESTED: "border-amber-200 bg-amber-50 text-amber-800",
  COMPLETED: "border-gray-200 bg-gray-50 text-gray-700",
};

export const STATUS_OPTIONS = [
  { value: "PENDING", label: STATUS_LABELS.PENDING },
  { value: "ACCEPTED", label: STATUS_LABELS.ACCEPTED },
  { value: "RESCHEDULE_REQUESTED", label: STATUS_LABELS.RESCHEDULE_REQUESTED },
  { value: "COMPLETED", label: STATUS_LABELS.COMPLETED },
];

export function statusLabel(status) {
  return STATUS_LABELS[status] || status || "";
}

export function statusBadge(status) {
  return STATUS_BADGES[status] || STATUS_BADGES.PENDING;
}

const MONTHS = [
  "Januari",
  "Februari",
  "Maret",
  "April",
  "Mei",
  "Juni",
  "Juli",
  "Agustus",
  "September",
  "Oktober",
  "November",
  "Desember",
];

/** "2026-11-10" → "10 November 2026". Returns null when there is no date. */
export function formatDate(value) {
  const match = /^(\d{4})-(\d{2})-(\d{2})/.exec(String(value || ""));
  if (!match) return null;
  const month = MONTHS[Number(match[2]) - 1];
  if (!month) return null;
  return `${Number(match[3])} ${month} ${match[1]}`;
}

/** "09:00:00" → "09.00". Returns null when there is no time. */
export function formatTime(value) {
  const match = /^(\d{2}):(\d{2})/.exec(String(value || ""));
  return match ? `${match[1]}.${match[2]}` : null;
}

/** "2026-11-08T12:34:56" → "8 November 2026, 12.34". */
export function formatDateTime(value) {
  const date = formatDate(value);
  if (!date) return null;
  const time = formatTime(String(value).slice(11));
  return time ? `${date}, ${time}` : date;
}

const WIB_PARTS = new Intl.DateTimeFormat("sv-SE", {
  timeZone: "Asia/Jakarta",
  year: "numeric",
  month: "2-digit",
  day: "2-digit",
  hour: "2-digit",
  minute: "2-digit",
  second: "2-digit",
  hourCycle: "h23",
});

/** The current time as a WIB wall-clock string, "2026-11-09T12:00:00". */
export function nowInWib() {
  return WIB_PARTS.format(new Date()).replace(" ", "T");
}

/** 18500000 → "Rp18.500.000". Returns null when the amount is unknown. */
export function formatPrice(value) {
  if (value === null || value === undefined || value === "") return null;
  const number = Number(value);
  if (Number.isNaN(number)) return null;
  return `Rp${number.toLocaleString("id-ID", { maximumFractionDigits: 0 })}`;
}

/** Joins the date and time of a booking for compact lists. */
export function formatSchedule(date, time) {
  const day = formatDate(date);
  const clock = formatTime(time);
  if (!day) return null;
  return clock ? `${day} · ${clock}` : day;
}
