const CONTROL_CHARACTERS = /[\u0000-\u001F\u007F-\u009F]/;

function hasCredentialSyntax(value: string): boolean {
  const authorityStart = value.indexOf("//");
  if (authorityStart < 0) return false;
  return value.slice(authorityStart + 2).split(/[\\/?#]/, 1)[0].includes("@");
}

function isLoopbackHostname(hostname: string): boolean {
  const normalized = hostname.toLowerCase().replace(/^\[|\]$/g, "").replace(/\.$/, "");
  if (normalized === "localhost" || normalized.endsWith(".localhost") || normalized === "::1") {
    return true;
  }
  const octets = normalized.split(".");
  return octets.length === 4
    && octets[0] === "127"
    && octets.every((octet) => /^\d{1,3}$/.test(octet) && Number(octet) <= 255);
}

export function normalizeEndpointURL(value: string): string {
  const trimmed = value.trim();
  if (!trimmed || value.length > 2048 || CONTROL_CHARACTERS.test(value)) {
    throw new Error("控制服务器地址包含无效字符。");
  }
  if (trimmed.includes("?") || trimmed.includes("#")) {
    throw new Error("控制服务器地址不能包含查询参数或片段。");
  }
  const schemeSeparator = trimmed.indexOf("://");
  const authority = schemeSeparator > 0
    ? trimmed.slice(schemeSeparator + 3).split(/[/?#]/, 1)[0]
    : "";
  if (!authority) {
    throw new Error("控制服务器地址缺少主机名。");
  }

  let parsed: URL;
  try {
    parsed = new URL(trimmed);
  } catch {
    throw new Error("控制服务器地址格式无效。");
  }
  if (!parsed.hostname || parsed.username || parsed.password || hasCredentialSyntax(trimmed)) {
    throw new Error("控制服务器地址不能包含凭据。");
  }

  const secure = parsed.protocol === "https:";
  const localDevelopment = parsed.protocol === "http:" && isLoopbackHostname(parsed.hostname);
  if (!secure && !localDevelopment) {
    throw new Error("非本机控制服务器必须使用 HTTPS。");
  }

  const pathname = parsed.pathname === "/" ? "" : parsed.pathname.replace(/\/+$/, "");
  return `${parsed.protocol}//${parsed.host}${pathname}`;
}
