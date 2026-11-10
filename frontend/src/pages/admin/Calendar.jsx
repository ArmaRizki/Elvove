import AdminPageHeader from "../../components/admin/AdminPageHeader";
import BookingCalendar from "../../components/admin/BookingCalendar";

// Kept as a direct URL (/admin/calendar) for existing bookmarks. The calendar
// itself lives in BookingCalendar and is also embedded in the Dashboard.
export default function Calendar() {
  return (
    <div className="min-w-0">
      <AdminPageHeader
        title="Kalender Pemesanan"
        description="Hanya menampilkan pemesanan yang telah diterima."
      />
      <BookingCalendar />
    </div>
  );
}
