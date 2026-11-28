import { useEffect, useState } from "react";
import { useSearchParams } from "react-router-dom";
import {
  CalendarCheck,
  CheckCircle2,
  ExternalLink,
  FolderPlus,
  Link2,
  LoaderCircle,
  Plug,
  RefreshCw,
  Save,
  Unplug,
  XCircle,
} from "lucide-react";
import { adminApi } from "../../services/adminApi";
import AdminPageHeader from "../../components/admin/AdminPageHeader";
import {
  adminCard,
  adminInput,
  adminLabel,
  adminNotice,
  adminPrimaryButton,
  adminSecondaryButton,
} from "../../components/admin/adminStyles";

// Why the OAuth callback came back without a connection. Shown once, on the
// page Google redirected to.
const callbackReasons = {
  state: "Sesi penyambungan tidak valid atau sudah kedaluwarsa. Coba hubungkan lagi.",
  exchange: "Google menolak penukaran kode. Periksa Client ID dan Client Secret.",
  not_configured: "Isi Client ID dan Client Secret terlebih dahulu.",
  save: "Token tidak dapat disimpan. Coba lagi.",
  encrypt: "Token tidak dapat dienkripsi di server.",
  secret: "Client Secret tersimpan tidak dapat dibaca. Isi ulang.",
};

function callbackNotice(params) {
  const result = params.get("drive");
  if (!result) return { type: "", text: "" };
  if (result === "connected") {
    return { type: "success", text: "Akun Google Drive berhasil tersambung." };
  }
  return {
    type: "error",
    text: callbackReasons[params.get("reason")] || "Penyambungan Google Drive gagal.",
  };
}

/**
 * Google Drive setup.
 *
 * <p>Two steps: the OAuth client (once, from Google Cloud Console) and the
 * consent that connects the account. After that every booking gets its own
 * folder with a webp and a "foto terpilih" subfolder, so the admin never pastes a
 * Drive link again.</p>
 */
export default function DriveSettings() {
  const [searchParams, setSearchParams] = useSearchParams();
  const [settings, setSettings] = useState(null);
  const [form, setForm] = useState({ clientId: "", clientSecret: "" });
  const [rootLink, setRootLink] = useState("");
  const [rootName, setRootName] = useState("ELVOVE Foto");
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState("");
  // Read once as the initial value: the query string is cleared below, and the
  // message has to outlive that.
  const [notice, setNotice] = useState(() => callbackNotice(searchParams));

  const load = () =>
    adminApi
      .getDriveSettings()
      .then(({ data }) => {
        setSettings(data);
        setForm((old) => ({ ...old, clientId: data.clientId || "" }));
        setRootName((old) => data.rootFolderName || old);
      })
      .catch((error) =>
        setNotice({
          type: "error",
          text: error.response?.data?.message || "Pengaturan Google Drive tidak dapat dimuat.",
        })
      )
      .finally(() => setLoading(false));

  useEffect(() => {
    load();
  }, []);

  // After the callback the state has to be re-read, and the outcome is taken
  // out of the address bar so a reload does not repeat it.
  useEffect(() => {
    if (!searchParams.get("drive")) return;
    load();
    setSearchParams({}, { replace: true });
  }, [searchParams, setSearchParams]);

  const run = async (key, action, success) => {
    setBusy(key);
    setNotice({ type: "", text: "" });
    try {
      const { data } = await action();
      if (data && typeof data === "object" && "connected" in data) setSettings(data);
      if (success) setNotice({ type: "success", text: success });
      return data;
    } catch (error) {
      setNotice({
        type: "error",
        text: error.response?.data?.message || "Tindakan gagal. Coba lagi.",
      });
      return null;
    } finally {
      setBusy("");
    }
  };

  const saveClient = (event) => {
    event.preventDefault();
    run("client", () => adminApi.saveDriveSettings(form), "Client OAuth tersimpan.");
  };

  // The consent has to run in a new tab: Google does not allow its pages in an
  // iframe, and the admin has to pick the Google account there. The window is
  // opened before the request: browsers block one opened after an await.
  const connect = async () => {
    const popup = window.open("about:blank", "_blank");
    if (popup) popup.opener = null;
    const data = await run("connect", () => adminApi.connectDrive());
    if (!data?.authUrl) {
      popup?.close();
      return;
    }
    if (popup) popup.location = data.authUrl;
    else window.open(data.authUrl, "_blank", "noopener");
  };

  const saveRoot = (event) => {
    event.preventDefault();
    run("root", () => adminApi.saveDriveRoot(rootLink.trim()), "Folder induk tersimpan.");
  };

  if (loading) return <p className="py-10 text-sm text-gray-500">Memuat pengaturan Google Drive...</p>;

  // A failed load leaves settings null; the page must say so instead of reading
  // fields off it.
  if (!settings) {
    return (
      <p className="py-10 text-sm text-gray-600">
        {notice.text || "Pengaturan Google Drive tidak dapat dimuat. Muat ulang halaman ini."}
      </p>
    );
  }

  const connected = Boolean(settings?.connected);

  return (
    <div className="min-w-0">
      <AdminPageHeader
        title="Google Drive"
        description="Hubungkan akun Google Drive agar folder foto klien dibuat otomatis setiap ada pemesanan."
      />

      {notice.text && (
        <div className={`mb-6 ${adminNotice(notice.type)}`} role="status">
          {notice.text}
        </div>
      )}

      <section className={`${adminCard} mb-6`}>
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div className="flex items-start gap-3">
            {connected ? (
              <CheckCircle2 size={20} className="mt-0.5 shrink-0 text-green-600" />
            ) : (
              <XCircle size={20} className="mt-0.5 shrink-0 text-gray-400" />
            )}
            <div>
              <h2 className="font-display text-lg text-gray-900">
                {connected ? "Tersambung" : "Belum tersambung"}
              </h2>
              <p className="mt-1 text-sm text-gray-600">
                {connected
                  ? `Akun ${settings.accountEmail || "Google"} menyimpan folder foto klien.`
                  : "Selama belum tersambung, pemesanan tetap masuk tetapi folder foto belum dibuat."}
              </p>
            </div>
          </div>
          {connected && (
            <button
              type="button"
              onClick={() => run("disconnect", () => adminApi.disconnectDrive(), "Google Drive diputuskan.")}
              disabled={busy === "disconnect"}
              className={adminSecondaryButton}
            >
              {busy === "disconnect" ? <LoaderCircle size={16} className="animate-spin" /> : <Unplug size={16} />}
              Putuskan
            </button>
          )}
        </div>

        {!settings.encryptionReady && (
          <p className="mt-4 border border-amber-200 bg-amber-50 p-3 text-xs text-amber-800">
            SMTP_ENCRYPTION_KEY belum diatur di server. Rahasia (Client Secret dan token) tidak dapat
            disimpan sampai kunci itu diisi.
          </p>
        )}
      </section>

      <form onSubmit={saveClient} className={`${adminCard} mb-6`}>
        <h2 className="mb-1 font-display text-lg text-gray-900">1. Client OAuth Google</h2>
        <p className="mb-4 text-sm text-gray-600">
          Buat OAuth client tipe <strong>Web application</strong> di Google Cloud Console, aktifkan
          Google Drive API, lalu isi{" "}
          <span className="font-mono text-xs">Authorized redirect URIs</span> dengan alamat di bawah
          dan tempel dua nilai berikut. Tipe <em>Desktop app</em> tidak bisa dipakai: tipe itu tidak
          menerima alamat balik <span className="font-mono text-xs">https://</span>.
        </p>
        <p className="mb-4 text-xs text-gray-500">
          Tidak perlu kartu kredit: pembuatan OAuth client dan pemakaian normal Drive API gratis.
        </p>

        <div className="mb-4 border border-gray-200 bg-gray-50 p-3">
          <p className={adminLabel}>REDIRECT URI YANG HARUS DIDAFTARKAN</p>
          <code className="mt-1 block break-all text-xs text-gray-800">{settings.redirectUri}</code>
        </div>

        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <div className="sm:col-span-2">
            <label className={adminLabel} htmlFor="drive-client-id">
              CLIENT ID
            </label>
            <input
              id="drive-client-id"
              className={adminInput}
              required
              value={form.clientId}
              onChange={(event) => setForm({ ...form, clientId: event.target.value })}
              placeholder="1234567890-abc.apps.googleusercontent.com"
              autoComplete="off"
            />
          </div>
          <div className="sm:col-span-2">
            <label className={adminLabel} htmlFor="drive-client-secret">
              CLIENT SECRET
            </label>
            <input
              id="drive-client-secret"
              className={adminInput}
              type="password"
              value={form.clientSecret}
              onChange={(event) => setForm({ ...form, clientSecret: event.target.value })}
              placeholder={
                settings.secretConfigured
                  ? "Tersimpan. Kosongkan bila tidak diganti."
                  : "Tempel Client Secret dari Google Cloud Console"
              }
              autoComplete="new-password"
            />
            <p className="mt-1 text-xs text-gray-500">
              Disimpan terenkripsi dan tidak pernah dikirim kembali ke browser.
            </p>
          </div>
        </div>

        <div className="mt-5 flex flex-wrap gap-2">
          <button type="submit" disabled={busy === "client"} className={adminPrimaryButton}>
            {busy === "client" ? <LoaderCircle size={16} className="animate-spin" /> : <Save size={16} />}
            Simpan Client
          </button>
          <button
            type="button"
            onClick={connect}
            disabled={busy === "connect" || !settings.secretConfigured}
            className={adminSecondaryButton}
          >
            {busy === "connect" ? <LoaderCircle size={16} className="animate-spin" /> : <Plug size={16} />}
            {connected ? "Hubungkan Ulang" : "Hubungkan Akun Google"}
          </button>
        </div>
      </form>

      <section className={`${adminCard} mb-6`}>
        <h2 className="mb-1 font-display text-lg text-gray-900">2. Folder Induk</h2>
        <p className="mb-4 text-sm text-gray-600">
          Semua folder klien dibuat di dalam folder ini. Buat sekali, lalu biarkan.
        </p>

        {settings.rootFolderId ? (
          <div className="mb-4 flex flex-wrap items-center gap-3 border border-green-200 bg-green-50 p-3 text-sm text-green-900">
            <Link2 size={16} className="shrink-0" />
            <span className="min-w-0 flex-1 truncate">
              {settings.rootFolderName || settings.rootFolderId}
            </span>
            <a
              href={`https://drive.google.com/drive/folders/${settings.rootFolderId}`}
              target="_blank"
              rel="noreferrer"
              className="inline-flex items-center gap-1 text-xs underline"
            >
              Buka <ExternalLink size={12} />
            </a>
          </div>
        ) : (
          <p className="mb-4 border border-amber-200 bg-amber-50 p-3 text-sm text-amber-800">
            Belum ada folder induk. Buat baru, atau tempel link folder Drive yang sudah ada.
          </p>
        )}

        <div className="flex flex-col gap-3">
          <div className="flex flex-col gap-2 sm:flex-row">
            <input
              className={adminInput}
              value={rootName}
              onChange={(event) => setRootName(event.target.value)}
              aria-label="Nama folder induk baru"
              placeholder="ELVOVE Foto"
            />
            <button
              type="button"
              onClick={() => run("create-root", () => adminApi.createDriveRoot(rootName), "Folder induk dibuat di My Drive.")}
              disabled={!connected || busy === "create-root"}
              className={`${adminSecondaryButton} shrink-0`}
            >
              {busy === "create-root" ? <LoaderCircle size={16} className="animate-spin" /> : <FolderPlus size={16} />}
              Buat Folder Induk
            </button>
          </div>

          <form onSubmit={saveRoot} className="flex flex-col gap-2 sm:flex-row">
            <input
              className={adminInput}
              value={rootLink}
              onChange={(event) => setRootLink(event.target.value)}
              placeholder="https://drive.google.com/drive/folders/..."
              aria-label="Link folder induk yang sudah ada"
            />
            <button
              type="submit"
              disabled={!connected || !rootLink.trim() || busy === "root"}
              className={`${adminSecondaryButton} shrink-0`}
            >
              {busy === "root" ? <LoaderCircle size={16} className="animate-spin" /> : <Link2 size={16} />}
              Pakai Folder Ini
            </button>
          </form>
        </div>
      </section>

      <section className={`${adminCard} mb-6`}>
        <h2 className="mb-1 font-display text-lg text-gray-900">3. Pemesanan yang Sudah Ada</h2>
        <p className="mb-4 text-sm text-gray-600">
          Pemesanan yang masuk sebelum Drive tersambung belum punya folder. Buat sekaligus di sini;
          pemesanan baru dibuat otomatis.
        </p>
        <button
          type="button"
          onClick={() =>
            run(
              "backfill",
              () => adminApi.backfillDriveFolders(),
              null
            ).then((data) => {
              if (data) {
                setNotice({
                  type: data.failed ? "error" : "success",
                  text: `${data.created} folder dibuat, ${data.failed} gagal, dari ${data.total} pemesanan.`,
                });
              }
            })
          }
          disabled={!connected || !settings.rootFolderId || busy === "backfill"}
          className={adminPrimaryButton}
        >
          {busy === "backfill" ? <LoaderCircle size={16} className="animate-spin" /> : <RefreshCw size={16} />}
          Buat Folder yang Belum Ada
        </button>
      </section>

      <section className={adminCard}>
        <h2 className="mb-1 font-display text-lg text-gray-900">4. Sinkron Kalender</h2>
        <p className="mb-4 text-sm text-gray-600">
          Menyamakan semua pemesanan bertanggal ke Google Calendar akun yang tersambung di atas.
          Pemesanan baru dan perubahan status tersinkron otomatis; tombol ini untuk menyusul yang belum.
        </p>
        <button
          type="button"
          onClick={() =>
            run("sync", () => adminApi.syncCalendar(), null).then((data) => {
              if (data) {
                setNotice({
                  type: data.failed ? "error" : "success",
                  text: `${data.synced} acara tersinkron, ${data.failed} gagal.`,
                });
              }
            })
          }
          disabled={!connected || busy === "sync"}
          className={adminPrimaryButton}
        >
          {busy === "sync" ? <LoaderCircle size={16} className="animate-spin" /> : <CalendarCheck size={16} />}
          Sinkronkan Kalender
        </button>
      </section>
    </div>
  );
}
