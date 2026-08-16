import { describe, expect, it } from "vitest";
import { validateLoginURL } from "./externalUrl";

describe("validateLoginURL", () => {
  it.each([
    ["https://login.example.com/register?token=abc#confirm", "https://login.example.com/register?token=abc#confirm"],
    ["http://localhost:8080/register", "http://localhost:8080/register"],
    ["http://127.0.0.42:8080/register", "http://127.0.0.42:8080/register"],
    ["http://[::1]:8080/register", "http://[::1]:8080/register"],
  ])("accepts a safe login URL: %s", (value, expected) => {
    expect(validateLoginURL(value)).toBe(expected);
  });

  it.each([
    "http://headscale.example.com/register",
    "http://127.0.0.1.evil.example/register",
    "http://[::2]/register",
    "https://user@example.com/register",
    "https://user:password@example.com/register",
    "https://@example.com/register",
    "javascript:alert(document.domain)",
    "file:///tmp/login.html",
    "https://",
    "https:///control",
    "/relative/login",
    " https://login.example.com/register",
    "https://login.example.com/register\n?token=abc",
    "https://login.example.com/register\u0000",
  ])("rejects an unsafe login URL: %s", (value) => {
    expect(() => validateLoginURL(value)).toThrow();
  });
});
