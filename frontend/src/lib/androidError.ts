import { createTranslator, type MessageKey } from "./i18n";

type Problem = { code: string; message?: string };
const messages: Record<string, MessageKey> = {
  "daemon-incompatible": "error.mobileCore",
  "daemon-unavailable": "error.mobileUnavailable",
  "daemon-unauthorized": "error.mobilePermission",
  "permission-denied": "error.mobilePermission",
  "control-unavailable": "error.mobileControl",
  "timeout": "error.mobileTimeout",
  "cancelled": "error.mobileCancelled",
  "internal": "common.operationFailed",
};

export function androidErrorMessage(problem?: Problem, fallback?: string): string {
  const t = createTranslator(document.documentElement.lang.startsWith("en") ? "en-US" : "zh-CN");
  const key = problem && messages[problem.code];
  if (key) return `${t(key)} (${problem!.code})`;
  return problem?.message || fallback || t("common.operationFailed");
}
