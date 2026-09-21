export function Alert({ tone = "error", children }: { tone?: "error" | "success" | "info"; children: React.ReactNode }) {
  const styles = {
    error: "border-red-200 bg-red-50 text-red-700",
    success: "border-emerald-200 bg-emerald-50 text-emerald-700",
    info: "border-navy-200 bg-navy-50 text-navy-800",
  }[tone];

  return <div className={`rounded-md border px-4 py-3 text-sm ${styles}`}>{children}</div>;
}
