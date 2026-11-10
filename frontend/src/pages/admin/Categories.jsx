import AdminResourceManager from "../../components/AdminResourceManager";
import { adminApi } from "../../services/adminApi";

export default function Categories() {
  return (
    <AdminResourceManager
      title="Kategori"
      columns={[{ key: "name", label: "Nama" }, { key: "slug", label: "Slug" }]}
      fields={[{ name: "name", label: "Nama", type: "text", required: true }]}
      emptyDefaults={{ name: "" }}
      api={{
        list: adminApi.getCategories,
        create: adminApi.createCategory,
        update: adminApi.updateCategory,
        delete: adminApi.deleteCategory,
      }}
    />
  );
}
