import { useEffect } from "react";
import { EditorContent, useEditor, useEditorState } from "@tiptap/react";
import StarterKit from "@tiptap/starter-kit";
import TextAlign from "@tiptap/extension-text-align";
import { FontSize, TextStyle } from "@tiptap/extension-text-style";
import {
  AlignCenter,
  AlignJustify,
  AlignLeft,
  AlignRight,
  Bold,
  Italic,
  Link2,
  List,
  ListOrdered,
  Quote,
  Redo2,
  RemoveFormatting,
  SeparatorHorizontal,
  Strikethrough,
  Underline,
  Undo2,
} from "lucide-react";
import { adminLabel } from "./adminStyles";

const FONT_SIZES = [
  { label: "Ukuran bawaan", value: "" },
  { label: "12 px", value: "12px" },
  { label: "14 px", value: "14px" },
  { label: "16 px", value: "16px" },
  { label: "18 px", value: "18px" },
  { label: "20 px", value: "20px" },
  { label: "24 px", value: "24px" },
  { label: "28 px", value: "28px" },
  { label: "32 px", value: "32px" },
];

const HEADINGS = [
  { label: "Teks biasa", value: 0 },
  { label: "Judul 1", value: 1 },
  { label: "Judul 2", value: 2 },
  { label: "Judul 3", value: 3 },
];

const ALIGNMENTS = [
  { value: "left", label: "Rata kiri", Icon: AlignLeft },
  { value: "center", label: "Rata tengah", Icon: AlignCenter },
  { value: "right", label: "Rata kanan", Icon: AlignRight },
  { value: "justify", label: "Rata kanan-kiri", Icon: AlignJustify },
];

function ToolButton({ label, active, disabled, onClick, children }) {
  return (
    <button
      type="button"
      title={label}
      aria-label={label}
      aria-pressed={active}
      disabled={disabled}
      // Keep the editor selection while a toolbar control is pressed.
      onMouseDown={(event) => event.preventDefault()}
      onClick={onClick}
      className={`inline-flex h-8 w-8 items-center justify-center border transition-colors disabled:cursor-not-allowed disabled:opacity-40 ${
        active
          ? "border-[#E11D48] bg-[#E11D48] text-white"
          : "border-transparent text-gray-600 hover:border-gray-200 hover:bg-gray-50"
      }`}
    >
      {children}
    </button>
  );
}

function Divider() {
  return <span className="mx-1 h-5 w-px self-center bg-gray-200" aria-hidden="true" />;
}

const selectClass =
  "h-8 border border-gray-200 bg-white px-2 text-xs text-gray-700 outline-none focus:border-[#E11D48]";

/**
 * Rich-text field for long page copy. It stores HTML; every page that shows the
 * content renders it through toHtml() in utils/richText.js.
 */
export default function RichTextEditor({ id, label, hint, value, onChange, minHeight = "16rem", columnBreak = false }) {
  const editor = useEditor({
    extensions: [
      StarterKit.configure({
        heading: { levels: [1, 2, 3] },
        link: {
          openOnClick: false,
          autolink: true,
          HTMLAttributes: { rel: "noopener noreferrer", target: "_blank" },
        },
      }),
      TextStyle,
      FontSize,
      TextAlign.configure({ types: ["heading", "paragraph"] }),
    ],
    content: value || "",
    editorProps: {
      attributes: {
        id,
        class: "rich-text px-4 py-3 outline-none",
        style: `min-height: ${minHeight}`,
        role: "textbox",
        "aria-multiline": "true",
        "aria-label": label,
      },
    },
    onUpdate: ({ editor: current }) => onChange(current.getHTML()),
  });

  // The stored value arrives after the first render on the admin pages, and it
  // is replaced after a save. Typing already keeps the two in sync, so this
  // only runs when the value came from somewhere else.
  useEffect(() => {
    if (!editor || value === editor.getHTML()) return;
    editor.commands.setContent(value || "", { emitUpdate: false });
  }, [editor, value]);

  const state = useEditorState({
    editor,
    selector: ({ editor: current }) => ({
      bold: current?.isActive("bold") ?? false,
      italic: current?.isActive("italic") ?? false,
      underline: current?.isActive("underline") ?? false,
      strike: current?.isActive("strike") ?? false,
      bulletList: current?.isActive("bulletList") ?? false,
      orderedList: current?.isActive("orderedList") ?? false,
      blockquote: current?.isActive("blockquote") ?? false,
      link: current?.isActive("link") ?? false,
      heading: [1, 2, 3].find((level) => current?.isActive("heading", { level })) ?? 0,
      align: ALIGNMENTS.find(({ value: alignment }) => current?.isActive({ textAlign: alignment }))?.value ?? "left",
      fontSize: current?.getAttributes("textStyle").fontSize ?? "",
      canUndo: current?.can().undo() ?? false,
      canRedo: current?.can().redo() ?? false,
    }),
  });

  if (!editor) return null;

  const active = state ?? {};
  const chain = () => editor.chain().focus();

  function handleLink() {
    const previous = editor.getAttributes("link").href || "";
    const href = window.prompt("Alamat tautan (kosongkan untuk menghapus)", previous || "https://");
    if (href === null) return;
    if (!href.trim()) {
      chain().extendMarkRange("link").unsetLink().run();
      return;
    }
    chain().extendMarkRange("link").setLink({ href: href.trim() }).run();
  }

  return (
    <div className="min-w-0">
      <span className={`${adminLabel} block`}>{label}</span>

      <div className="mt-1 border border-gray-200 focus-within:border-[#E11D48]">
        <div className="flex flex-wrap items-center gap-0.5 border-b border-gray-200 bg-gray-50 p-1.5">
          <ToolButton label="Urungkan" disabled={!active.canUndo} onClick={() => chain().undo().run()}>
            <Undo2 size={15} />
          </ToolButton>
          <ToolButton label="Ulangi" disabled={!active.canRedo} onClick={() => chain().redo().run()}>
            <Redo2 size={15} />
          </ToolButton>

          <Divider />

          <select
            aria-label="Gaya paragraf"
            title="Gaya paragraf"
            className={selectClass}
            value={String(active.heading ?? 0)}
            onChange={(event) => {
              const level = Number(event.target.value);
              if (level === 0) chain().setParagraph().run();
              else chain().setHeading({ level }).run();
            }}
          >
            {HEADINGS.map(({ label: text, value: level }) => (
              <option key={level} value={level}>{text}</option>
            ))}
          </select>

          <select
            aria-label="Ukuran huruf"
            title="Ukuran huruf"
            className={selectClass}
            value={active.fontSize || ""}
            onChange={(event) => {
              const size = event.target.value;
              if (size) chain().setFontSize(size).run();
              else chain().unsetFontSize().run();
            }}
          >
            {FONT_SIZES.map(({ label: text, value: size }) => (
              <option key={size} value={size}>{text}</option>
            ))}
          </select>

          {columnBreak && (
            <ToolButton
              label="Pisah kolom"
              onClick={() => chain().setHorizontalRule().run()}
            >
              <SeparatorHorizontal size={15} />
            </ToolButton>
          )}

          <Divider />

          <ToolButton label="Tebal" active={active.bold} onClick={() => chain().toggleBold().run()}>
            <Bold size={15} />
          </ToolButton>
          <ToolButton label="Miring" active={active.italic} onClick={() => chain().toggleItalic().run()}>
            <Italic size={15} />
          </ToolButton>
          <ToolButton label="Garis bawah" active={active.underline} onClick={() => chain().toggleUnderline().run()}>
            <Underline size={15} />
          </ToolButton>
          <ToolButton label="Coret" active={active.strike} onClick={() => chain().toggleStrike().run()}>
            <Strikethrough size={15} />
          </ToolButton>

          <Divider />

          {ALIGNMENTS.map(({ value: alignment, label: text, Icon }) => (
            <ToolButton
              key={alignment}
              label={text}
              active={active.align === alignment}
              onClick={() => chain().setTextAlign(alignment).run()}
            >
              <Icon size={15} />
            </ToolButton>
          ))}

          <Divider />

          <ToolButton label="Daftar butir" active={active.bulletList} onClick={() => chain().toggleBulletList().run()}>
            <List size={15} />
          </ToolButton>
          <ToolButton label="Daftar bernomor" active={active.orderedList} onClick={() => chain().toggleOrderedList().run()}>
            <ListOrdered size={15} />
          </ToolButton>
          <ToolButton label="Kutipan" active={active.blockquote} onClick={() => chain().toggleBlockquote().run()}>
            <Quote size={15} />
          </ToolButton>

          <Divider />

          <ToolButton label="Tautan" active={active.link} onClick={handleLink}>
            <Link2 size={15} />
          </ToolButton>
          <ToolButton
            label="Hapus format"
            onClick={() => chain().unsetAllMarks().clearNodes().run()}
          >
            <RemoveFormatting size={15} />
          </ToolButton>
        </div>

        <EditorContent editor={editor} />
      </div>

      {hint && <p className="mt-2 text-xs text-gray-500">{hint}</p>}
    </div>
  );
}
