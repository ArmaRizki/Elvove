import { useRef, useState } from "react";
import { LoaderCircle, Mail, ReceiptText, Save } from "lucide-react";
import { adminApi } from "../../services/adminApi";
import { adminInput, adminLabel, adminNotice, adminSecondaryButton } from "./adminStyles";
import {
  PAYMENT_OPTIONS,
  formatDateTime,
  nowInWib,
  paymentBadge,
  paymentLabel,
} from "./adminFormat";

/**
 * Payment state, the delivery link and the two closing e-mails of a booking.
 *
 * <p>The order matters: the confirmation asks for the DP, the settlement notice
 * asks for the balance once the DP is recorded, and the invoice closes the order
 * with the transaction details and the link to the edited photos. Every send is
 * manual, and each button stays disabled until the booking has the data that
 * message needs.</p>
 */
export default function BookingPayment({ booking, onUpdated }) {
  const [saving, setSaving] = useState(false);
  const [savingLink, setSavingLink] = useState(false);
  const [sending, setSending] = useState("");
  const [notice, setNotice] = useState({ type: "", text: "" });
  const [link, setLink] = useState(booking?.deliveryUrl || "");
  // Synchronous guard: two clicks in the same tick both see the ref, unlike
  // React state, which only updates after the next render.
  const inFlight = useRef(false);

  if (!booking) return null;

  const email = (booking.email || "").trim();
  // The API only mails a booking whose date has been agreed, so both closing
  // messages start from ACCEPTED.
  const confirmed = booking.status === "ACCEPTED" || booking.status === "COMPLETED";
  const completed = booking.status === "COMPLETED";
  const paid = booking.paymentStatus === "PAID";
  const downPaid = booking.paymentStatus === "DOWN_PAYMENT_PAID" || paid;
  const deliveryUrl = (booking.deliveryUrl || "").trim();
  const invoiceCount = Number(booking.invoiceEmailCount) || 0;
  const settlementCount = Number(booking.settlementEmailCount) || 0;
  const invoiceSent = formatDateTime(booking.invoiceEmailSentAt);
  const settlementSent = formatDateTime(booking.settlementEmailSentAt);

  const savePayment = async (paymentStatus) => {
    if (saving || paymentStatus === (booking.paymentStatus || "UNPAID")) return;
    setSaving(true);
    setNotice({ type: "", text: "" });

    try {
      const { data } = await adminApi.updateBookingPayment(booking.id, paymentStatus);
      onUpdated?.(data);
      setNotice({
        type: "success",
        text: `Status pembayaran diubah menjadi ${paymentLabel(paymentStatus)}.`,
      });
    } catch (error) {
      setNotice({
        type: "error",
        text: error.response?.data?.message || "Status pembayaran gagal diperbarui.",
      });
    } finally {
      setSaving(false);
    }
  };

  const saveLink = async () => {
    if (savingLink) return;
    setSavingLink(true);
    setNotice({ type: "", text: "" });

    try {
      const { data } = await adminApi.updateBookingDelivery(booking.id, link.trim());
      onUpdated?.(data);
      setNotice({
        type: "success",
        text: link.trim()
          ? "Link foto tersimpan dan akan disertakan pada email invoice."
          : "Link foto dikosongkan.",
      });
    } catch (error) {
      setNotice({
        type: "error",
        text: error.response?.data?.message || "Link foto gagal disimpan.",
      });
    } finally {
      setSavingLink(false);
    }
  };

  const send = async (kind) => {
    if (inFlight.current) return;
    inFlight.current = true;
    setSending(kind);
    setNotice({ type: "", text: "" });

    const isInvoice = kind === "invoice";
    const label = isInvoice ? "invoice" : "pelunasan";
    const sentAtKey = isInvoice ? "invoiceEmailSentAt" : "settlementEmailSentAt";
    const countKey = isInvoice ? "invoiceEmailCount" : "settlementEmailCount";
    const sentCount = isInvoice ? invoiceCount : settlementCount;

    try {
      const { data } = isInvoice
        ? await adminApi.sendBookingInvoice(booking.id)
        : await adminApi.sendBookingSettlement(booking.id);
      setNotice({
        type: "success",
        text: data?.message || `Email ${label} berhasil dikirim ke ${email}.`,
      });
      onUpdated?.({
        [sentAtKey]: data?.[sentAtKey] ?? nowInWib(),
        [countKey]: data?.[countKey] ?? sentCount + 1,
      });
    } catch (error) {
      setNotice({
        type: "error",
        text:
          error.response?.data?.message ||
          `Email ${label} gagal dikirim. Silakan coba lagi.`,
      });
    } finally {
      inFlight.current = false;
      setSending("");
    }
  };

  const busy = saving || savingLink || sending !== "";
  const linkChanged = link.trim() !== deliveryUrl;

  return (
    <div className="space-y-4">
      <div>
        <label className={adminLabel} htmlFor="booking-payment">
          STATUS PEMBAYARAN
        </label>
        <select
          id="booking-payment"
          value={booking.paymentStatus || "UNPAID"}
          disabled={saving}
          onChange={(event) => savePayment(event.target.value)}
          className="mt-1 w-full border border-gray-200 bg-transparent px-3 py-2.5 text-sm text-gray-900 focus:border-[#E11D48] outline-none disabled:opacity-60 sm:max-w-xs"
        >
          {PAYMENT_OPTIONS.map((option) => (
            <option key={option.value} value={option.value}>
              {option.label}
            </option>
          ))}
        </select>
        <p className="mt-2 flex flex-wrap items-center gap-2 text-xs text-gray-500">
          <span className={`inline-flex border px-2 py-0.5 ${paymentBadge(booking.paymentStatus)}`}>
            {paymentLabel(booking.paymentStatus)}
          </span>
          {saving ? (
            <span className="flex items-center gap-2">
              <LoaderCircle size={13} className="animate-spin" />
              Menyimpan status pembayaran...
            </span>
          ) : (
            <span>Menandai lunas ikut mencatat DP sebagai diterima.</span>
          )}
        </p>
      </div>

      <div>
        <label className={adminLabel} htmlFor="booking-delivery">
          LINK GOOGLE DRIVE FOTO EDIT
        </label>
        <div className="mt-1 flex flex-col gap-2 sm:flex-row">
          <input
            id="booking-delivery"
            type="url"
            inputMode="url"
            placeholder="https://drive.google.com/drive/folders/..."
            value={link}
            onChange={(event) => setLink(event.target.value)}
            className={adminInput}
          />
          <button
            type="button"
            onClick={saveLink}
            disabled={!linkChanged || busy}
            aria-busy={savingLink}
            className={`${adminSecondaryButton} shrink-0`}
          >
            {savingLink ? (
              <LoaderCircle size={16} className="animate-spin" />
            ) : (
              <Save size={16} />
            )}
            Simpan Link
          </button>
        </div>
        <p className="mt-1 text-xs text-gray-500">
          Link ini disertakan pada email invoice. Isi setelah foto hasil edit selesai
          diunggah ke Google Drive.
        </p>
      </div>

      {!email ? (
        <p className={adminNotice("error")}>
          Pemesanan ini tidak memiliki alamat email klien, sehingga email pelunasan dan
          invoice tidak dapat dikirim.
        </p>
      ) : (
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <div className="min-w-0">
            <button
              type="button"
              onClick={() => send("settlement")}
              disabled={!confirmed || !downPaid || paid || busy}
              aria-busy={sending === "settlement"}
              className={`${adminSecondaryButton} w-full`}
            >
              {sending === "settlement" ? (
                <LoaderCircle size={16} className="animate-spin" />
              ) : (
                <Mail size={16} />
              )}
              {sending === "settlement"
                ? "Mengirim email..."
                : settlementCount > 0
                  ? "Kirim Ulang Email Pelunasan"
                  : "Kirim Email Pelunasan"}
            </button>
            <p className="mt-1 text-xs text-gray-500">
              {paid
                ? "Pemesanan sudah lunas, tidak ada sisa yang perlu ditagih."
                : !downPaid
                  ? "Tandai DP dibayar dulu sebelum menagih sisa pembayaran."
                  : settlementSent
                    ? `Terakhir dikirim pada ${settlementSent}.`
                    : "Menagih sisa pembayaran dengan tenggat H-1 sesi."}
            </p>
          </div>

          <div className="min-w-0">
            <button
              type="button"
              onClick={() => send("invoice")}
              disabled={!completed || !paid || !deliveryUrl || busy}
              aria-busy={sending === "invoice"}
              className={`${adminSecondaryButton} w-full`}
            >
              {sending === "invoice" ? (
                <LoaderCircle size={16} className="animate-spin" />
              ) : (
                <ReceiptText size={16} />
              )}
              {sending === "invoice"
                ? "Mengirim invoice..."
                : invoiceCount > 0
                  ? "Kirim Ulang Email Invoice"
                  : "Kirim Email Invoice"}
            </button>
            <p className="mt-1 text-xs text-gray-500">
              {!completed
                ? "Tersedia setelah pemesanan selesai."
                : !paid
                  ? "Tandai lunas dulu sebelum mengirim invoice."
                  : !deliveryUrl
                    ? "Isi dan simpan link Google Drive foto edit dulu."
                    : invoiceSent
                      ? `Terakhir dikirim pada ${invoiceSent}.`
                      : "Berisi rincian transaksi dan link foto edit."}
            </p>
          </div>
        </div>
      )}

      {notice.text && (
        <p className={adminNotice(notice.type)} role="status">
          {notice.text}
        </p>
      )}
    </div>
  );
}
