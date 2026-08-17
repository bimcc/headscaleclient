import { Check, ChevronDown, Settings } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import type { Endpoint, LoginProfile } from "../lib/contracts";
import { useI18n } from "../lib/i18n";

export function AccountMenu({
  profiles,
  endpoints,
  activeProfileId,
  switching,
  onSwitch,
  onManage,
}: {
  profiles: LoginProfile[];
  endpoints: Endpoint[];
  activeProfileId: string | null;
  switching: boolean;
  onSwitch: (profileId: string) => Promise<void>;
  onManage: () => void;
}) {
  const { t } = useI18n();
  const [open, setOpen] = useState(false);
  const rootRef = useRef<HTMLDivElement>(null);
  const menuRef = useRef<HTMLDivElement>(null);
  const activeProfile = profiles.find((profile) => profile.id === activeProfileId) ?? null;

  useEffect(() => {
    if (!open) return;
    const closeOnOutsideClick = (event: PointerEvent) => {
      if (!rootRef.current?.contains(event.target as Node)) setOpen(false);
    };
    const closeOnEscape = (event: KeyboardEvent) => {
      if (event.key !== "Escape") return;
      event.preventDefault();
      setOpen(false);
      rootRef.current?.querySelector<HTMLButtonElement>(".account-button")?.focus();
    };
    document.addEventListener("pointerdown", closeOnOutsideClick);
    document.addEventListener("keydown", closeOnEscape);
    const focusFrame = window.requestAnimationFrame(() => {
      const selected = menuRef.current?.querySelector<HTMLElement>('[aria-checked="true"]');
      const first = menuRef.current?.querySelector<HTMLElement>('[role="menuitemradio"], [role="menuitem"]');
      (selected ?? first)?.focus();
    });
    return () => {
      window.cancelAnimationFrame(focusFrame);
      document.removeEventListener("pointerdown", closeOnOutsideClick);
      document.removeEventListener("keydown", closeOnEscape);
    };
  }, [open]);

  const moveFocus = (direction: 1 | -1) => {
    const items = Array.from(
      menuRef.current?.querySelectorAll<HTMLElement>('[role="menuitemradio"]:not(:disabled), [role="menuitem"]:not(:disabled)') ?? [],
    );
    if (items.length === 0) return;
    const current = items.indexOf(document.activeElement as HTMLElement);
    const next = current < 0 ? 0 : (current + direction + items.length) % items.length;
    items[next]?.focus();
  };

  return (
    <div className="account-menu" ref={rootRef}>
      <button
        className="account-button"
        type="button"
        aria-haspopup="menu"
        aria-expanded={open}
        aria-label={t("account.switchCurrent", { account: activeProfile?.account ?? t("common.notLoggedIn") })}
        onClick={() => setOpen((value) => !value)}
      >
        <span className="account-avatar">{(activeProfile?.account ?? "?").slice(0, 1)}</span>
        <span className="account-label truncate">{activeProfile?.account ?? t("common.notLoggedIn")}</span>
        <ChevronDown aria-hidden="true" size={16} />
      </button>

      {open && (
        <div
          className="account-popover"
          role="menu"
          aria-label={t("account.switchMenu")}
          ref={menuRef}
          onKeyDown={(event) => {
            if (event.key === "ArrowDown" || event.key === "ArrowUp") {
              event.preventDefault();
              moveFocus(event.key === "ArrowDown" ? 1 : -1);
            }
          }}
        >
          <div className="account-popover-label">{t("account.networks")}</div>
          <div className="account-popover-profiles">
            {profiles.map((profile) => {
              const active = profile.id === activeProfileId;
              const endpoint = endpoints.find((item) => item.id === profile.endpointId);
              return (
                <button
                  className="account-option"
                  type="button"
                  role="menuitemradio"
                  aria-checked={active}
                  disabled={switching}
                  key={profile.id}
                  onClick={async () => {
                    try {
                      if (!active) await onSwitch(profile.id);
                      setOpen(false);
                    } catch {
                      // App owns the user-facing error toast; keep this menu open for retry.
                    }
                  }}
                >
                  <span className="account-avatar">{profile.account.slice(0, 1)}</span>
                  <span className="account-option-copy truncate">
                    <strong className="truncate">{profile.account}</strong>
                    <small className="truncate">{endpoint?.name ?? t("account.unknownEndpoint")}</small>
                  </span>
                  {active && <Check aria-label={t("account.current")} size={17} />}
                </button>
              );
            })}
          </div>
          <button
            className="account-manage"
            type="button"
            role="menuitem"
            onClick={() => {
              setOpen(false);
              onManage();
            }}
          >
            <Settings aria-hidden="true" size={17} />
            {t("account.manage")}
          </button>
        </div>
      )}
    </div>
  );
}
