const CONTROL_CHARACTERS = /[\u0000-\u001F\u007F-\u009F]/;

function isLoopbackHostname(hostname: string): boolean {
  const normalized = hostname.toLowerCase().replace(/^\[|\]$/g, "");
  if (normalized === "localhost" || normalized === "::1") return true;

  const octets = normalized.split(".");
  return octets.length === 4
    && octets[0] === "127"
    && octets.every((octet) => /^\d{1,3}$/.test(octet) && Number(octet) <= 255);
}

function hasCredentialSyntax(value: string): boolean {
  const authorityStart = value.indexOf("//");
  if (authorityStart < 0) return false;

  const authority = value.slice(authorityStart + 2).split(/[\\/?#]/, 1)[0];
  return authority.includes("@");
}

export function validateLoginURL(value: string): string {
  if (!value || value !== value.trim() || CONTROL_CHARACTERS.test(value)) {
    throw new Error("登录地址包含不允许的字符。");
  }
  const schemeSeparator = value.indexOf("://");
  const authority = schemeSeparator > 0
    ? value.slice(schemeSeparator + 3).split(/[\\/?#]/, 1)[0]
    : "";
  if (!authority) {
    throw new Error("登录地址缺少主机名。");
  }

  let parsed: URL;
  try {
    parsed = new URL(value);
  } catch {
    throw new Error("登录地址格式无效。");
  }

  if (!parsed.hostname) {
    throw new Error("登录地址缺少主机名。");
  }
  if (parsed.username || parsed.password || hasCredentialSyntax(value)) {
    throw new Error("登录地址不能包含用户名或密码。");
  }

  const secure = parsed.protocol === "https:";
  const localDevelopment = parsed.protocol === "http:" && isLoopbackHostname(parsed.hostname);
  if (!secure && !localDevelopment) {
    throw new Error("登录地址必须使用 HTTPS；HTTP 仅允许本机回环地址。");
  }

  return parsed.href;
}
