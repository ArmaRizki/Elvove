import { useEffect, useState } from "react";
import { Phone, Mail, MapPin, MessageCircle } from "lucide-react";
import { publicApi } from "../services/publicApi";
import SectionHeading from "../components/SectionHeading";
import LoadingState from "../components/LoadingState";

export default function Contact() {
  const [contact, setContact] = useState(null);

  useEffect(() => {
    publicApi.getContact().then((res) => setContact(res.data));
  }, []);

  if (!contact) return <LoadingState />;

  const formatWhatsApp = (number) => {
    if (!number) return "";

    let cleaned = number.replace(/\D/g, "");

    if (cleaned.startsWith("0")) {
      cleaned = "62" + cleaned.substring(1);
    }

    return cleaned;
  };

  const whatsappUrl = contact.whatsappNumber
    ? `https://wa.me/${formatWhatsApp(contact.whatsappNumber)}`
    : null;

  return (
    <div className="pt-28 sm:pt-32 pb-20 sm:pb-24 max-w-7xl mx-auto px-5 sm:px-6">
      <SectionHeading as="h1" title="Hubungi Kami" />

      <div className="grid md:grid-cols-2 gap-10 md:gap-14">
        <div className="space-y-6">
          {contact.phone && (
            <div className="flex items-center gap-4">
              <Phone size={18} aria-hidden="true" className="text-[#E11D48] shrink-0" />
              <a
                href={`tel:${contact.phone}`}
                className="text-gray-700 hover:text-[#E11D48] transition-colors break-all"
              >
                {contact.phone}
              </a>
            </div>
          )}

          {whatsappUrl && (
            <div className="flex items-center gap-4">
              <MessageCircle size={18} aria-hidden="true" className="text-[#25D366] shrink-0" />
              <a
                href={whatsappUrl}
                target="_blank"
                rel="noopener noreferrer"
                className="text-gray-700 hover:text-[#25D366] transition-colors"
              >
                Hubungi via WhatsApp
              </a>
            </div>
          )}

          {contact.email && (
            <div className="flex items-center gap-4">
              <Mail size={18} aria-hidden="true" className="text-[#E11D48] shrink-0" />
              <a
                href={`mailto:${contact.email}`}
                className="text-gray-700 hover:text-[#E11D48] transition-colors break-all"
              >
                {contact.email}
              </a>
            </div>
          )}

          {contact.address && (
            <div className="flex items-start gap-4">
              <MapPin size={18} aria-hidden="true" className="text-[#E11D48] mt-1 shrink-0" />
              <p className="text-gray-700 leading-relaxed">{contact.address}</p>
            </div>
          )}
        </div>

        {contact.googleMapLink && (
          <iframe
            src={contact.googleMapLink}
            className="w-full h-64 sm:h-80 border border-gray-200"
            loading="lazy"
            title="Lokasi studio"
          />
        )}
      </div>
    </div>
  );
}
