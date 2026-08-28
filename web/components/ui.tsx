import { ReactNode } from "react";

export function Panel({ children, className = "" }: { children: ReactNode; className?: string }) {
  return (
    <div
      className={`bg-panel border border-border rounded-xl p-5 shadow-sm animate-fade-in ${className}`}
    >
      {children}
    </div>
  );
}

export function Button({
  children,
  variant = "primary",
  className = "",
  ...rest
}: {
  children: ReactNode;
  variant?: "primary" | "secondary" | "danger";
  className?: string;
} & React.ButtonHTMLAttributes<HTMLButtonElement>) {
  const styles = {
    primary: "bg-accent text-white hover:bg-accent-hover",
    secondary: "bg-transparent border border-border text-text hover:bg-accent-soft hover:border-accent",
    danger: "bg-danger text-white hover:opacity-90",
  }[variant];
  return (
    <button
      className={`inline-flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-semibold transition-all active:scale-[.97] ${styles} ${className}`}
      {...rest}
    >
      {children}
    </button>
  );
}

export function Input(props: React.InputHTMLAttributes<HTMLInputElement>) {
  return (
    <input
      {...props}
      className={`w-full px-3 py-2 bg-input-bg border border-border rounded-lg text-sm text-text focus:outline-none focus:border-accent focus:ring-2 focus:ring-accent-soft transition-colors ${props.className ?? ""}`}
    />
  );
}

export function Textarea(props: React.TextareaHTMLAttributes<HTMLTextAreaElement>) {
  return (
    <textarea
      {...props}
      className={`w-full px-3 py-2 bg-input-bg border border-border rounded-lg text-sm text-text font-mono focus:outline-none focus:border-accent focus:ring-2 focus:ring-accent-soft transition-colors ${props.className ?? ""}`}
    />
  );
}

export function Label({ children }: { children: ReactNode }) {
  return <label className="block mt-3 mb-1.5 text-xs font-medium text-muted">{children}</label>;
}

export function Badge({ children, tone = "default" }: { children: ReactNode; tone?: "default" | "ok" | "danger" }) {
  const styles = {
    default: "bg-border text-muted",
    ok: "bg-ok-soft text-ok badge-ok-dot",
    danger: "bg-danger-soft text-danger",
  }[tone];
  return (
    <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-[11px] font-semibold ${styles}`}>
      {children}
    </span>
  );
}

export function Flash({ message, kind }: { message: string; kind: "ok" | "error" }) {
  if (!message) return null;
  const styles = kind === "ok" ? "bg-ok-soft text-ok" : "bg-danger-soft text-danger";
  return <div className={`px-4 py-2.5 rounded-lg text-sm mb-5 animate-fade-in ${styles}`}>{message}</div>;
}

export function TableWrap({ children }: { children: ReactNode }) {
  return <div className="overflow-x-auto rounded-xl">{children}</div>;
}

export function H1({ icon, children }: { icon: ReactNode; children: ReactNode }) {
  return (
    <h1 className="flex items-center gap-2.5 text-xl font-bold mb-6 tracking-tight">
      {icon}
      {children}
    </h1>
  );
}

export function H2({ children }: { children: ReactNode }) {
  return <h2 className="mt-7 mb-2.5 text-sm font-semibold text-muted">{children}</h2>;
}
