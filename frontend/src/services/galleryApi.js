import api from "./api";

// The client photo-selection page. The token comes from the e-mail link and is
// the only credential, so these calls carry no Authorization header.
export const galleryApi = {
  getGallery: (token) => api.get(`/api/gallery/${token}`),
  photoUrl: (token, fileId) => `/api/gallery/${token}/photos/${fileId}`,
  saveSelection: (token, fileIds) => api.put(`/api/gallery/${token}/selection`, { fileIds }),
  submitSelection: (token) => api.post(`/api/gallery/${token}/submit`),
};
