import { useCallback, useEffect, useRef, useState } from "react";

/**
 * Loads the list an admin page shows, and remembers whether it is still loading.
 *
 * Every admin page had its own copy of the same three lines: a `loading` flag,
 * a `load()` that turns the flag on, fetches, and turns it off again, and a
 * `useEffect` that calls it once on mount. The copies drifted — some reported a
 * failed load to the admin, some swallowed it and left an empty table that read
 * as "no records yet".
 *
 * `fetch` must be stable: define it with `useCallback`, or it will re-run the
 * initial load on every render.
 *
 * Returns:
 *   data       what `fetch` returned, or `null` before the first answer
 *   loading    true while a load is running, including the first one
 *   error      the message of the last failed load, cleared when a load starts
 *   reload     runs the load again; the pages call it after saving or deleting
 *
 * Only the newest load may write `data` and `loading`: an admin who saves twice
 * in a row must not see the first answer arrive last and win.
 */
export default function useAdminList(fetch) {
  // `fetch` answers with the value to keep; useAdminList stores it as the last
  // load's answer and nothing more. It never compares two answers to decide
  // whether the data changed — a page that wants identity-stable data (see
  // Bookings) keeps its own copy — so the pages can hand each render the same
  // list they already show without triggering another load.
  const [data, setData] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  const latest = useRef(0);
  const mounted = useRef(true);
  useEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
    };
  }, []);

  const reload = useCallback(async () => {
    const run = ++latest.current;
    setLoading(true);
    setError("");
    try {
      const result = await fetch();
      // A page that navigated away, or a load the admin has since replaced,
      // must not set state.
      if (!mounted.current || run !== latest.current) return;
      setData(result);
    } catch (err) {
      if (!mounted.current || run !== latest.current) return;
      setError(err.response?.data?.message || "Gagal memuat data. Coba muat ulang halaman ini.");
    } finally {
      if (mounted.current && run === latest.current) setLoading(false);
    }
  }, [fetch]);

  useEffect(() => {
    reload();
  }, [reload]);

  return { data, loading, error, reload };
}
