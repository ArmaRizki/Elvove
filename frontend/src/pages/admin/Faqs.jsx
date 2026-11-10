import AdminResourceManager from "../../components/AdminResourceManager";
import { adminApi } from "../../services/adminApi";

export default function Faqs() {
  return (
    <AdminResourceManager
      title="Pertanyaan Umum"
      columns={[{ key: "question", label: "Pertanyaan" }]}
      fields={[
        { name: "question", label: "Pertanyaan", type: "text", required: true },
        { name: "answer", label: "Jawaban", type: "textarea", required: true },
        { name: "displayOrder", label: "Urutan tampil", type: "number" },
      ]}
      emptyDefaults={{ question: "", answer: "", displayOrder: 0 }}
      api={{
        list: adminApi.getFaqs,
        create: adminApi.createFaq,
        update: adminApi.updateFaq,
        delete: adminApi.deleteFaq,
      }}
    />
  );
}
