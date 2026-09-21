import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { ApiRequestError, authApi, candidateApi, setAccessToken, getAccessToken } from "@/lib/api";

function jsonResponse(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } });
}

describe("API client error handling", () => {
  beforeEach(() => {
    setAccessToken(null);
    vi.restoreAllMocks();
  });
  afterEach(() => {
    setAccessToken(null);
  });

  it("resolves with data on a successful envelope", async () => {
    global.fetch = vi.fn().mockResolvedValue(
      jsonResponse(200, { success: true, data: { id: 1, email: "a@b.com" } })
    );
    const user = await authApi.me();
    expect(user).toEqual({ id: 1, email: "a@b.com" });
  });

  it("throws ApiRequestError with the server's code/message on a failure envelope", async () => {
    global.fetch = vi.fn().mockResolvedValue(
      jsonResponse(422, {
        success: false,
        error: { code: "VALIDATION_ERROR", message: "email is required", details: null },
      })
    );

    await expect(authApi.me()).rejects.toMatchObject({
      status: 422,
      code: "VALIDATION_ERROR",
      message: "email is required",
    });
  });

  it("throws a generic ApiRequestError when the response body isn't valid JSON", async () => {
    global.fetch = vi.fn().mockResolvedValue(new Response("not json", { status: 500 }));
    await expect(authApi.me()).rejects.toBeInstanceOf(ApiRequestError);
  });

  it("attaches the in-memory access token as a Bearer header", async () => {
    setAccessToken("test-token-123");
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse(200, { success: true, data: {} }));
    global.fetch = fetchMock;

    await authApi.me();

    const [, options] = fetchMock.mock.calls[0];
    expect((options.headers as Record<string, string>)["Authorization"]).toBe("Bearer test-token-123");
  });

  it("sends credentials: include on every request (for the httpOnly refresh cookie)", async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse(200, { success: true, data: {} }));
    global.fetch = fetchMock;

    await authApi.me();

    const [, options] = fetchMock.mock.calls[0];
    expect(options.credentials).toBe("include");
  });

  it("on a 401, attempts a silent refresh once and retries the original request", async () => {
    let call = 0;
    const fetchMock = vi.fn().mockImplementation((url: string) => {
      call++;
      if (call === 1) {
        // first call: the original request, unauthorized
        return Promise.resolve(jsonResponse(401, { success: false, error: { code: "UNAUTHORIZED", message: "expired" } }));
      }
      if (String(url).includes("/auth/refresh")) {
        // second call: the silent refresh succeeds
        return Promise.resolve(jsonResponse(200, { success: true, data: { access_token: "new-token-456" } }));
      }
      // third call: retried original request, now succeeds
      return Promise.resolve(jsonResponse(200, { success: true, data: { id: 1 } }));
    });
    global.fetch = fetchMock;

    const result = await authApi.me();

    expect(result).toEqual({ id: 1 });
    expect(fetchMock).toHaveBeenCalledTimes(3);
    expect(getAccessToken()).toBe("new-token-456");
  });

  it("clears the access token if silent refresh fails, and surfaces the original 401", async () => {
    setAccessToken("stale-token");
    const fetchMock = vi.fn().mockImplementation((url: string) => {
      if (String(url).includes("/auth/refresh")) {
        return Promise.resolve(jsonResponse(401, { success: false, error: { code: "UNAUTHORIZED", message: "no valid session" } }));
      }
      return Promise.resolve(jsonResponse(401, { success: false, error: { code: "UNAUTHORIZED", message: "expired" } }));
    });
    global.fetch = fetchMock;

    await expect(authApi.me()).rejects.toMatchObject({ status: 401 });
    expect(getAccessToken()).toBeNull();
  });

  it("shares one refresh operation across concurrent unauthorized requests", async () => {
    let refreshCalls = 0;
    let protectedCalls = 0;
    const fetchMock = vi.fn().mockImplementation((url: string) => {
      if (String(url).includes("/auth/refresh")) {
        refreshCalls += 1;
        return new Promise((resolve) => {
          setTimeout(() => resolve(jsonResponse(200, { success: true, data: { access_token: "shared-token" } })), 10);
        });
      }
      protectedCalls += 1;
      if (protectedCalls <= 2) {
        return Promise.resolve(jsonResponse(401, { success: false, error: { code: "UNAUTHORIZED", message: "expired" } }));
      }
      return Promise.resolve(jsonResponse(200, { success: true, data: { ok: true } }));
    });
    global.fetch = fetchMock;

    const results = await Promise.all([authApi.me(), authApi.me()]);

    expect(results).toEqual([{ ok: true }, { ok: true }]);
    expect(refreshCalls).toBe(1);
  });

  it("sends multipart form data for resume upload without a Content-Type header (browser sets boundary)", async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse(201, { success: true, data: { id: 1, status: "parsed" } }));
    global.fetch = fetchMock;

    const file = new File(["hello world"], "resume.txt", { type: "text/plain" });
    await candidateApi.uploadResume(file);

    const [, options] = fetchMock.mock.calls[0];
    expect(options.body).toBeInstanceOf(FormData);
    expect((options.headers as Record<string, string>)["Content-Type"]).toBeUndefined();
  });
});
