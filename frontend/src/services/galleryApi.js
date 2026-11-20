import api from "./api";

// The client photo-selection page. The token comes from the e-mail link and is
// the only credential, so these calls carry no Authorization header.
export const galleryApi = {
  getGallery: (token) => api.get(`/api/gallery/${token}`),
  photoUrl: (token, fileId) => `/api/gallery/${token}/photos/${fileId}`,
  // driveThumbnail is the picture Drive itself keeps for a photo: already
  // resized, so the page fills in a second instead of waiting for the original to
  // be converted. It is only a preview; a pick still copies the original.
  driveThumbnail: (fileId, width) =>
    `https://drive.google.com/thumbnail?id=${encodeURIComponent(fileId)}&sz=w${width}`,
  saveSelection: (token, fileIds) => api.put(`/api/gallery/${token}/selection`, { fileIds }),
  submitSelection: (token) => api.post(`/api/gallery/${token}/submit`),
};
