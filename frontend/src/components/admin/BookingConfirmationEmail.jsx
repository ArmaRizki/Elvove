import { useRef, useState } from "react";
import { LoaderCircle, Mail } from "lucide-react";
import { adminApi } from "../../services/adminApi";
import { adminNotice, adminPrimaryButton } from "./adminStyles";
import { formatDateTime, nowInWib } from "./adminFormat";

/**
 * Explicit confirmation e-mail step. It only appears for accepted bookings and
 * only sends when the admin presses the button, so a status change never
 * notifies the client on its own.
 *
 * <p>The button becomes "Kirim Ulang Email Konfirmasi" once a delivery has been
 * recorded, and the last successful send is shown underneath.</p>
 */
export default function BookingConfirmationEmail({ booking, onSent }) {
  const [sending, setSending] = useState(false);
  const [notice, setNotice] = useState({ type: "", text: "" });
  // Synchronous guard: two clicks in the same tick both see the ref, unlike
  // React state, which only updates after the next render.
  const inFlight = useRef(false);

  if (!booking || booking.status !== "ACCEPTED") return null;

  const email = (booking.email || "").trim();
  const sentCount = Number(booking.confirmationEmailCount) || 0;
  const lastSent = formatDateTime(booking.confirmationEmailSentAt);

  if (!email) {
    return (
      <p className={adminNotice("error")}>
        Pemesanan ini tidak memiliki alamat email klien, sehingga konfirmasi tidak dapat dikirim.
      </p>
    );
  }

  const send = async () => {
    if (inFlight.current) return;
    inFlight.current = true;
    setSending(true);
    setNotice({ type: "", text: "" });

    try {
      const { data } = await adminApi.sendBookingConfirmation(booking.id);
      setNotice({
        type: "success",
        text: data?.message || `Email konfirmasi berhasil dikirim ke ${email}.`,
      });
      onSent?.({
        confirmationEmailSentAt: data?.confirmationEmailSentAt ?? nowInWib(),
        confirmationEmailCount: data?.confirmationEmailCount ?? sentCount + 1,
      });
    } catch (error) {
      setNotice({
        type: "error",
        text:
          error.response?.data?.message ||
          "Email konfirmasi gagal dikirim. Silakan coba lagi.",
      });
    } finally {
      inFlight.current = false;
      setSending(false);
    }
  };

  return (
    <div className="space-y-2">
      <button
        type="button"
        onClick={send}
        disabled={sending}
        aria-busy={sending}
        className={`${adminPrimaryButton} w-full sm:w-auto`}
      >
        {sending ? (
          <LoaderCircle size={16} className="animate-spin" />
        ) : (
          <Mail size={16} />
        )}
        {sending
          ? "Mengirim email..."
          : sentCount > 0
            ? "Kirim Ulang Email Konfirmasi"
            : "Kirim Email Konfirmasi"}
      </button>

      <p className="text-xs text-gray-500">
        Email berisi detail pemesanan akan dikirim ke {email}.
        {lastSent
          ? ` Terakhir berhasil dikirim pada ${lastSent}.`
          : " Email konfirmasi belum pernah dikirim."}
      </p>

      {notice.text && (
        <p className={adminNotice(notice.type)} role="status">
          {notice.text}
        </p>
      )}
    </div>
  );
}
