export function getImageUrl(path) {
  if (!path) return "";

  // Jika URL berasal dari backend lama, gunakan path relatif.
  if (path.startsWith("http://") || path.startsWith("https://")) {
    try {
      const url = new URL(path);

      if (url.pathname.startsWith("/uploads/")) {
        return url.pathname + url.search;
      }

      return path;
    } catch {
      return path;
    }
  }

  return path.startsWith("/") ? path : `/${path}`;
}
