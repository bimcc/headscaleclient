import { describe, expect, it } from "vitest";
import { normalizeEndpointURL } from "./endpointUrl";

describe("normalizeEndpointURL", () => {
  it.each([
    ["HTTPS://Headscale.Example:443/", "https://headscale.example"],
    ["https://headscale.example:8443/control/", "https://headscale.example:8443/control"],
    [" http://127.0.0.1:8080/ ", "http://127.0.0.1:8080"],
    ["http://dev.localhost:8080", "http://dev.localhost:8080"],
    ["http://[::1]:8080", "http://[::1]:8080"],
  ])("normalizes %s", (value, expected) => {
    expect(normalizeEndpointURL(value)).toBe(expected);
  });

  it.each([
    "http://headscale.example",
    "https://user:secret@headscale.example",
    "https://@headscale.example",
    "https://headscale.example?tenant=a",
    "https://headscale.example#fragment",
    "file:///tmp/control",
    "https:///control",
    "https://headscale.example\n/control",
  ])("rejects %s", (value) => {
    expect(() => normalizeEndpointURL(value)).toThrow();
  });
});
