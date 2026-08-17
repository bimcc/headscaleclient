import type { LucideIcon } from "lucide-react";
import { AlertCircle, Inbox, LoaderCircle } from "lucide-react";
import type { ButtonHTMLAttributes, ReactNode } from "react";
import { useI18n } from "../lib/i18n";

export type Tone = "neutral" | "success" | "warning" | "danger";

export function StatusBadge({
  tone = "neutral",
  children,
}: {
  tone?: Tone;
  children: ReactNode;
}) {
  return <span className={`status-badge tone-${tone}`}>{children}</span>;
}

export function IconButton({
  label,
  icon: Icon,
  ...props
}: ButtonHTMLAttributes<HTMLButtonElement> & {
  label: string;
  icon: LucideIcon;
}) {
  return (
    <button className="icon-button" type="button" title={label} aria-label={label} {...props}>
      <Icon aria-hidden="true" size={18} strokeWidth={1.8} />
    </button>
  );
}

export function Toggle({
  label,
  checked,
  disabled,
  onChange,
}: {
  label: string;
  checked: boolean;
  disabled?: boolean;
  onChange: (checked: boolean) => void;
}) {
  return (
    <button
      className="toggle"
      type="button"
      role="switch"
      aria-label={label}
      aria-checked={checked}
      disabled={disabled}
      onClick={() => onChange(!checked)}
    >
      <span className="toggle-thumb" />
    </button>
  );
}

export function SettingRow({
  title,
  description,
  control,
  nested = false,
  disabled = false,
}: {
  title: string;
  description?: string;
  control: ReactNode;
  nested?: boolean;
  disabled?: boolean;
}) {
  return (
    <div
      className={`setting-row${nested ? " is-nested" : ""}${disabled ? " is-disabled" : ""}`}
      aria-disabled={disabled || undefined}
    >
      <div className="setting-copy">
        <span className="setting-title">{title}</span>
        {description && <span className="setting-description">{description}</span>}
      </div>
      <div className="setting-control">{control}</div>
    </div>
  );
}

export function EmptyState({
  title,
  message,
  action,
}: {
  title: string;
  message: string;
  action?: ReactNode;
}) {
  return (
    <div className="empty-state">
      <Inbox aria-hidden="true" size={26} strokeWidth={1.5} />
      <strong>{title}</strong>
      <span>{message}</span>
      {action}
    </div>
  );
}

export function ErrorState({ message, onRetry }: { message: string; onRetry: () => void }) {
  const { t } = useI18n();
  return (
    <div className="state-page" role="alert">
      <AlertCircle aria-hidden="true" size={30} />
      <h1>{t("error.loadTitle")}</h1>
      <p>{message}</p>
      <button className="button primary" type="button" onClick={onRetry}>
        {t("common.retry")}
      </button>
    </div>
  );
}

export function LoadingState() {
  const { t } = useI18n();
  return (
    <div className="loading-shell" aria-label={t("common.loading")} aria-busy="true">
      <div className="loading-nav" />
      <div className="loading-main">
        <div className="skeleton skeleton-title" />
        <div className="skeleton skeleton-banner" />
        <div className="skeleton skeleton-row" />
        <div className="skeleton skeleton-row short" />
      </div>
      <span className="sr-only">
        <LoaderCircle />{t("common.loading")}
      </span>
    </div>
  );
}

export function Modal({
  title,
  children,
  onClose,
}: {
  title: string;
  children: ReactNode;
  onClose: () => void;
}) {
  const { t } = useI18n();
  return (
    <div className="modal-backdrop" role="presentation" onMouseDown={onClose}>
      <section
        className="modal"
        role="dialog"
        aria-modal="true"
        aria-labelledby="modal-title"
        onMouseDown={(event) => event.stopPropagation()}
      >
        <header className="modal-header">
          <h2 id="modal-title">{title}</h2>
          <button className="button quiet" type="button" onClick={onClose}>
            {t("common.cancel")}
          </button>
        </header>
        {children}
      </section>
    </div>
  );
}
