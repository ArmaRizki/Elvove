import { formatDate, formatDateTime, formatTime } from "./adminFormat";
import { formatPrice } from "../../utils/format";

function Field({ label, value, breakAll = false, children }) {
  return (
    <div className="min-w-0">
      <dt className="text-[10px] tracking-widest text-gray-500">{label.toUpperCase()}</dt>
      <dd className={`mt-1 text-sm text-gray-900 ${breakAll ? "break-all" : "break-words"}`}>
        {children ?? value}
      </dd>
    </div>
  );
}

function Section({ title, children }) {
  return (
    <section className="p-4 sm:p-5">
      <h3 className="text-[10px] tracking-widest text-gray-500">{title.toUpperCase()}</h3>
      <div className="mt-3">{children}</div>
    </section>
  );
}

/**
 * The booking's facts only: who, when, what, and the notes. Everything the
 * action cards below already show — status, payment state, e-mail history,
 * the Drive links — is left out, so the popup never says the same thing twice.
 */
export default function BookingDetails({ booking }) {
  if (!booking) return null;

  const createdAt = formatDateTime(booking.createdAt);
  const downPaidAt = formatDate(booking.downPaymentPaidAt);
  const settledAt = formatDate(booking.settlementPaidAt);

  return (
    <div className="divide-y divide-gray-200">
      <Section title="Klien & acara">
        <dl className="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <Field label="Nama" value={booking.name || "—"} />
          <Field label="Email" breakAll value={booking.email || "—"} />
          <Field label="Telepon" value={booking.phone || "—"} />
          <Field label="Lokasi" value={booking.eventLocation || "—"} />
          <Field label="Tanggal acara" value={formatDate(booking.eventDate) || "—"} />
          <Field label="Waktu sesi" value={formatTime(booking.eventTime) || "—"} />
        </dl>
      </Section>

      <Section title="Paket">
        <dl className="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <Field label="Nama paket" value={booking.packageTitle || "—"} />
          <Field label="Harga" value={formatPrice(booking.packagePrice) || "—"} />
        </dl>
      </Section>

      <Section title="Catatan klien">
        {booking.message ? (
          <p className="whitespace-pre-line break-words text-sm text-gray-900">{booking.message}</p>
        ) : (
          <p className="text-sm text-gray-500">—</p>
        )}
      </Section>

      <Section title="Pencatatan">
        <dl className="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <Field label="Dibuat pada" value={createdAt || "—"} />
          {downPaidAt && <Field label="DP dibayar pada" value={downPaidAt} />}
          {settledAt && <Field label="Lunas pada" value={settledAt} />}
        </dl>
      </Section>
    </div>
  );
}
