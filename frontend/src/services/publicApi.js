import api from "./api";

export const publicApi = {
  getAbout: () => api.get("/api/public/about"),
  getTerms: () => api.get("/api/public/terms"),
  getContact: () => api.get("/api/public/contact"),
  getSocialLinks: () => api.get("/api/public/social-links"),
  getCategories: () => api.get("/api/public/categories"),
  getPhotos: (params) => api.get("/api/public/photos", { params }),
  getPackages: () => api.get("/api/public/packages"),
  createBooking: (data) => api.post("/api/bookings", data),
};
