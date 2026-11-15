// Indonesian labels and formatters shared by the admin panel, so statuses,
// dates and prices read the same way everywhere.
import { formatDateDisplay } from "../../utils/bookingTime";

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

// Payment points of a booking: nothing yet, the down payment, or the whole
// amount. The server derives this from the two payment dates.
export const PAYMENT_LABELS = {
  UNPAID: "Belum bayar",
  DOWN_PAYMENT_PAID: "DP dibayar",
  PAID: "Lunas",
};

export const PAYMENT_OPTIONS = [
  { value: "UNPAID", label: "Belum bayar" },
  { value: "DOWN_PAYMENT_PAID", label: "DP 50% dibayar" },
  { value: "PAID", label: "Lunas" },
];

const PAYMENT_BADGES = {
  UNPAID: "border-amber-200 bg-amber-50 text-amber-800",
  DOWN_PAYMENT_PAID: "border-sky-200 bg-sky-50 text-sky-800",
  PAID: "border-green-200 bg-green-50 text-green-800",
};

export function paymentLabel(paymentStatus) {
  return PAYMENT_LABELS[paymentStatus] || PAYMENT_LABELS.UNPAID;
}

export function paymentBadge(paymentStatus) {
  return PAYMENT_BADGES[paymentStatus] || PAYMENT_BADGES.UNPAID;
}

/**
 * "2026-11-10" or "2026-11-10T09:00:00" → "10/11/2026", the same dd/mm/yyyy
 * the booking form shows. Returns null when there is no date.
 */
export function formatDate(value) {
  return formatDateDisplay(String(value || "").slice(0, 10)) || null;
}

/** "09:00:00" → "09:00". Returns null when there is no time. */
export function formatTime(value) {
  const match = /^(\d{2}):(\d{2})/.exec(String(value || ""));
  return match ? `${match[1]}:${match[2]}` : null;
}

/** "2026-11-08T12:34:56" → "08/11/2026, 12:34". */
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

/** Joins the date and time of a booking for compact lists. */
export function formatSchedule(date, time) {
  const day = formatDate(date);
  const clock = formatTime(time);
  if (!day) return null;
  return clock ? `${day} · ${clock}` : day;
}
