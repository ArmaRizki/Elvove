
import { lazy, Suspense } from "react";
import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom";
import { AuthProvider } from "./context/AuthContext";
import ProtectedRoute from "./components/ProtectedRoute";
import PublicLayout from "./components/PublicLayout";
import AdminLayout from "./components/AdminLayout";
import LoadingState from "./components/LoadingState";

// Public pages
const Home = lazy(() => import("./pages/Home"));
const About = lazy(() => import("./pages/About"));
const Portfolio = lazy(() => import("./pages/Portfolio"));
const Packages = lazy(() => import("./pages/Packages"));
const LatestGallery = lazy(() => import("./pages/LatestGallery"));
const Booking = lazy(() => import("./pages/Booking"));
const Contact = lazy(() => import("./pages/Contact"));
const Privacy = lazy(() => import("./pages/Privacy"));
const Terms = lazy(() => import("./pages/Terms"));
const TermsAndConditions = lazy(() => import("./pages/TermsAndConditions"));

// Admin pages
const Login = lazy(() => import("./pages/admin/Login"));
const Dashboard = lazy(() => import("./pages/admin/Dashboard"));
const AdminPhotos = lazy(() => import("./pages/admin/Photos"));
const AdminCategories = lazy(() => import("./pages/admin/Categories"));
const AdminPackages = lazy(() => import("./pages/admin/Packages"));
const AdminBookings = lazy(() => import("./pages/admin/Bookings"));
const AdminCalendar = lazy(() => import("./pages/admin/Calendar"));
const AdminFaqs = lazy(() => import("./pages/admin/Faqs"));
const AboutEditor = lazy(() => import("./pages/admin/AboutEditor"));
const ContactEditor = lazy(() => import("./pages/admin/ContactEditor"));
const SocialLinksEditor = lazy(() => import("./pages/admin/SocialLinksEditor"));
const TermsEditor = lazy(() => import("./pages/admin/TermsEditor"));
const SmtpSettings = lazy(() => import("./pages/admin/SmtpSettings"));

export default function App() {
  return (
    <BrowserRouter>
      <AuthProvider>
        <Suspense fallback={<LoadingState />}>
          <Routes>
            {/* Public site */}
            <Route element={<PublicLayout />}>
              <Route path="/" element={<Home />} />
              <Route path="/about" element={<About />} />
              <Route path="/portfolio" element={<Portfolio />} />
              <Route path="/packages" element={<Packages />} />
              <Route path="/latest" element={<LatestGallery />} />
              <Route path="/booking" element={<Booking />} />
              <Route path="/contact" element={<Contact />} />
              <Route path="/privacy" element={<Privacy />} />
              <Route path="/terms" element={<Terms />} />
              <Route path="/terms-and-conditions" element={<TermsAndConditions />} />
            </Route>

            {/* Admin */}
            <Route path="/admin/login" element={<Login />} />

            <Route
              path="/admin"
              element={
                <ProtectedRoute>
                  <AdminLayout />
                </ProtectedRoute>
              }
            >
              <Route index element={<Navigate to="dashboard" replace />} />
              <Route path="dashboard" element={<Dashboard />} />
              <Route path="photos" element={<AdminPhotos />} />
              <Route path="categories" element={<AdminCategories />} />
              <Route path="packages" element={<AdminPackages />} />
              <Route path="bookings" element={<AdminBookings />} />
              <Route path="calendar" element={<AdminCalendar />} />
              <Route path="faqs" element={<AdminFaqs />} />
              <Route path="about" element={<AboutEditor />} />
              <Route path="contact" element={<ContactEditor />} />
              <Route path="social-links" element={<SocialLinksEditor />} />
              <Route path="terms" element={<TermsEditor />} />
              <Route path="smtp" element={<SmtpSettings />} />
            </Route>
          </Routes>
        </Suspense>
      </AuthProvider>
    </BrowserRouter>
  );
}
