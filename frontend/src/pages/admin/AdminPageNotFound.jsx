import { Link } from "react-router-dom";
import AdminPageHeader from "../../components/admin/AdminPageHeader";
import { adminPrimaryButton } from "../../components/admin/adminStyles";

/**
 * The admin page for an address that does not exist.
 *
 * It sits inside AdminLayout, so an admin who mistypes a menu address keeps the
 * sidebar and can go back to a real page. Like the public 404 it answers 200;
 * see pages/PageNotFound.jsx for why the status is nginx's business.
 */
export default function AdminPageNotFound() {
  return (
    <div>
      <AdminPageHeader
        title="Halaman tidak ada"
        description="Alamat ini bukan bagian dari panel admin."
        meta="404"
      />
      <div className="border border-gray-200 bg-white p-4 text-sm text-gray-700 sm:p-6">
        <p>Periksa alamatnya, atau kembali ke dasbor untuk memilih menu dari daftar.</p>
        <div className="mt-4 flex flex-wrap gap-2">
          <Link to="/admin/dashboard" className={adminPrimaryButton}>
            Ke dasbor
          </Link>
        </div>
      </div>
    </div>
  );
}
