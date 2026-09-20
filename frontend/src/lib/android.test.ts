import { afterEach, describe, expect, it, vi } from "vitest";
import { androidCall, androidSubscribe } from "./android";

afterEach(() => { delete window.HeadscaleAndroid; vi.useRealTimers(); });
describe("Android bridge", () => {
  it("correlates out-of-order responses and preserves native errors", async () => {
    const requests: { id: string }[] = [];
    window.HeadscaleAndroid = { postMessage: (raw) => requests.push(JSON.parse(raw)) };
    const first = androidCall("GetSnapshot"); const second = androidCall("Logout");
    const rejected = expect(second).rejects.toThrow("permission revoked");
    window.HeadscaleAndroid.onmessage!({ data: JSON.stringify({ id: requests[1].id, response: { error: "permission revoked" } }) });
    window.HeadscaleAndroid.onmessage!({ data: JSON.stringify({ id: requests[0].id, response: { result: { source: "native" } } }) });
    await expect(first).resolves.toEqual({ source: "native" }); await rejected;
  });
  it("times out requests without silently switching to demo data", async () => {
    vi.useFakeTimers(); window.HeadscaleAndroid = { postMessage: () => {} };
    const request = androidCall("SetConnection", [true]); const rejected = expect(request).rejects.toThrow();
    await vi.advanceTimersByTimeAsync(60001); await rejected;
  });
  it("refreshes on foreground return and stops listening when unmounted", async () => {
    const getSnapshot = vi.fn().mockResolvedValue({ source: "native" }); const onSnapshot = vi.fn();
    const stop = androidSubscribe(getSnapshot)(onSnapshot, vi.fn());
    window.dispatchEvent(new Event("headscale:resume")); await Promise.resolve();
    expect(getSnapshot).toHaveBeenCalledOnce(); expect(onSnapshot).toHaveBeenCalledOnce();
    stop(); window.dispatchEvent(new Event("headscale:resume")); expect(getSnapshot).toHaveBeenCalledOnce();
  });
});
