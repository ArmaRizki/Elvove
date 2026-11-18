import { useRef, useState } from "react";
import { ExternalLink, FolderOpen, Images, LoaderCircle, Mail, ReceiptText } from "lucide-react";
import { adminApi } from "../../services/adminApi";
import { adminLabel, adminNotice, adminSecondaryButton } from "./adminStyles";
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
 * <p>The order matters: the confirmation asks for the DP, the selection notice
 * hands the client the page where the photos to edit are chosen, the settlement
 * notice asks for the balance once the DP is recorded, and the invoice closes the
 * order with the transaction details and the link to the edited photos. Every
 * send is manual, and each button stays disabled until the booking has the data
 * that message needs.</p>
 *
 * <p>The delivery link is not typed in: it is the booking's "foto edit"
 * folder on Google Drive, filled in when the folder is created.</p>
 */
export default function BookingPayment({ booking, onUpdated }) {
  const [saving, setSaving] = useState(false);
  const [sending, setSending] = useState("");
  const [notice, setNotice] = useState({ type: "", text: "" });
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
  const selectionCount = Number(booking.selectionEmailCount) || 0;
  const invoiceSent = formatDateTime(booking.invoiceEmailSentAt);
  const settlementSent = formatDateTime(booking.settlementEmailSentAt);
  const selectionSent = formatDateTime(booking.selectionEmailSentAt);
  const rawFolder = (booking.rawFolderId || "").trim();

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

  const send = async (kind) => {
    if (inFlight.current) return;
    inFlight.current = true;
    setSending(kind);
    setNotice({ type: "", text: "" });

    const sentAtKey = `${kind}EmailSentAt`;
    const countKey = `${kind}EmailCount`;
    const sentCount =
      kind === "invoice" ? invoiceCount : kind === "selection" ? selectionCount : settlementCount;
    const label = kind === "invoice" ? "invoice" : kind === "selection" ? "pilihan foto" : "pelunasan";

    try {
      const { data } =
        kind === "invoice"
          ? await adminApi.sendBookingInvoice(booking.id)
          : kind === "selection"
            ? await adminApi.sendBookingSelection(booking.id)
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

  const busy = saving || sending !== "";

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
        <p className={adminLabel}>LINK FOTO EDIT (OTOMATIS)</p>
        {deliveryUrl ? (
          <div className="mt-1 flex flex-wrap items-center gap-3 border border-green-200 bg-green-50 p-3 text-sm text-green-900">
            <FolderOpen size={16} className="shrink-0" />
            <span className="min-w-0 flex-1 truncate font-mono text-xs">{deliveryUrl}</span>
            <a
              href={deliveryUrl}
              target="_blank"
              rel="noreferrer"
              className="inline-flex items-center gap-1 text-xs underline"
            >
              Buka <ExternalLink size={12} />
            </a>
          </div>
        ) : (
          <p className="mt-1 border border-amber-200 bg-amber-50 p-3 text-sm text-amber-800">
            Folder foto edit belum ada. Tekan <strong>Periksa</strong> pada folder foto klien
            di atas untuk membuatnya.
          </p>
        )}
        <p className="mt-1 text-xs text-gray-500">
          Mengarah ke subfolder <span className="font-mono">foto edit</span> dan ikut
          terkirim di email invoice. Unggah hasil edit ke folder itu.
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
              onClick={() => send("selection")}
              disabled={!confirmed || !rawFolder || busy}
              aria-busy={sending === "selection"}
              className={`${adminSecondaryButton} w-full`}
            >
              {sending === "selection" ? (
                <LoaderCircle size={16} className="animate-spin" />
              ) : (
                <Images size={16} />
              )}
              {sending === "selection"
                ? "Mengirim email..."
                : selectionCount > 0
                  ? "Kirim Ulang Email Pilihan Foto"
                  : "Kirim Email Pilihan Foto"}
            </button>
            <p className="mt-1 text-xs text-gray-500">
              {!rawFolder
                ? "Folder foto klien belum dibuat, jadi belum ada foto yang bisa dipilih."
                : selectionSent
                  ? `Terakhir dikirim pada ${selectionSent}. Kirim sebelum email pelunasan.`
                  : "Langkah pertama setelah sesi: klien memilih foto yang ingin diedit."}
            </p>
          </div>

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
                    : "Menagih sisa pembayaran setelah sesi foto."}
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
                    ? "Folder foto edit belum dibuat, jadi link foto edit belum ada."
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
