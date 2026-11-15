import {
  formatDate,
  formatDateTime,
  formatTime,
  paymentBadge,
  paymentLabel,
  statusBadge,
  statusLabel,
} from "./adminFormat";
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
 * Every piece of booking data the admin API returns, in one readable layout.
 * Package name and price are the values stored on the booking itself, so they
 * stay correct after the public package list is edited.
 */
export default function BookingDetails({ booking }) {
  if (!booking) return null;

  const packageName = booking.packageTitle || "Belum tercatat";
  const packagePrice = formatPrice(booking.packagePrice);
  const downPaidAt = formatDate(booking.downPaymentPaidAt);
  const settledAt = formatDate(booking.settlementPaidAt);
  const scheduleDate = formatDate(booking.eventDate);
  const scheduleTime = formatTime(booking.eventTime);
  const createdAt = formatDateTime(booking.createdAt);
  const updatedAt = formatDateTime(booking.updatedAt);
  const sentAt = formatDateTime(booking.confirmationEmailSentAt);
  const sentCount = Number(booking.confirmationEmailCount) || 0;
  const emailStatus =
    sentCount > 0
      ? sentAt
        ? `Terkirim ${sentCount} kali, terakhir ${sentAt}`
        : `Terkirim ${sentCount} kali`
      : "Belum dikirim";
  // Invoice and settlement are separate manual steps, so each keeps its own
  // line; a delivery count without a date only happens if the send stamped one
  // and not the other.
  const invoiceCount = Number(booking.invoiceEmailCount) || 0;
  const invoiceSentAt = formatDateTime(booking.invoiceEmailSentAt);
  const invoiceStatus =
    invoiceCount > 0
      ? invoiceSentAt
        ? `Terkirim ${invoiceCount} kali, terakhir ${invoiceSentAt}`
        : `Terkirim ${invoiceCount} kali`
      : "Belum dikirim";
  const settlementCount = Number(booking.settlementEmailCount) || 0;
  const settlementSentAt = formatDateTime(booking.settlementEmailSentAt);
  const settlementStatus =
    settlementCount > 0
      ? settlementSentAt
        ? `Terkirim ${settlementCount} kali, terakhir ${settlementSentAt}`
        : `Terkirim ${settlementCount} kali`
      : "Belum dikirim";

  return (
    <div className="divide-y divide-gray-200">
      <Section title="Data klien">
        <dl className="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <Field label="Nama" value={booking.name || "Belum tercatat"} />
          <Field label="Email" breakAll value={booking.email || "Belum tercatat"} />
          <Field label="Telepon" value={booking.phone || "Belum tercatat"} />
        </dl>
      </Section>

      <Section title="Detail acara">
        <dl className="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <Field label="Lokasi" value={booking.eventLocation || "Belum diisi"} />
          <Field label="Tanggal acara" value={scheduleDate || "Belum tercatat"} />
          <Field label="Waktu sesi" value={scheduleTime || "Belum ditentukan"} />
        </dl>
      </Section>

      <Section title="Paket yang dipilih">
        <dl className="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <Field label="Nama paket" value={packageName} />
          <Field label="Harga paket" value={packagePrice || "Belum tercatat"} />
        </dl>
        <p className="mt-3 text-xs text-gray-500">
          Nama dan harga paket tersimpan pada saat pemesanan dibuat, sehingga tetap sama
          meskipun paket di halaman publik diubah atau dihapus.
        </p>
      </Section>

      <Section title="Catatan klien">
        {booking.message ? (
          <p className="whitespace-pre-line break-words text-sm text-gray-900">{booking.message}</p>
        ) : (
          <p className="text-sm text-gray-500">Tidak ada catatan dari klien.</p>
        )}
      </Section>

      <Section title="Status dan administrasi">
        <dl className="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <div className="min-w-0">
            <dt className="text-[10px] tracking-widest text-gray-500">STATUS PEMESANAN</dt>
            <dd className="mt-1">
              <span className={`inline-flex border px-2 py-1 text-xs ${statusBadge(booking.status)}`}>
                {statusLabel(booking.status)}
              </span>
            </dd>
          </div>
          <div className="min-w-0">
            <dt className="text-[10px] tracking-widest text-gray-500">STATUS PEMBAYARAN</dt>
            <dd className="mt-1">
              <span
                className={`inline-flex border px-2 py-1 text-xs ${paymentBadge(booking.paymentStatus)}`}
              >
                {paymentLabel(booking.paymentStatus)}
              </span>
            </dd>
          </div>
          {downPaidAt && <Field label="DP dibayar pada" value={downPaidAt} />}
          {settledAt && <Field label="Lunas pada" value={settledAt} />}
          {booking.deliveryUrl && (
            <Field label="Link foto edit" breakAll value={booking.deliveryUrl} />
          )}
          <Field label="Dibuat pada" value={createdAt || "Belum tercatat"} />
          <Field label="Email konfirmasi" value={emailStatus} />
          <Field label="Email pelunasan" value={settlementStatus} />
          <Field label="Email invoice" value={invoiceStatus} />
          {updatedAt && updatedAt !== createdAt ? (
            <Field label="Terakhir diperbarui" value={updatedAt} />
          ) : null}
        </dl>
      </Section>
    </div>
  );
}
