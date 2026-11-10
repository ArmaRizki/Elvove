export default function LoadingState({ label = "Memuat" }) {
  return (
    <div className="flex items-center justify-center py-24 text-gray-700 text-sm tracking-widest">
      <span className="animate-pulse">{label}&hellip;</span>
    </div>
  );
}
