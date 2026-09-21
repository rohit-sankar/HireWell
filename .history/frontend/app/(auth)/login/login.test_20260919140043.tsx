import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { AuthProvider } from "@/providers/auth-provider";
import LoginPage from "@/app/(auth)/login/page";

const pushMock = vi.fn();
vi.mock("next/navigation", () => ({
  useRouter: () => ({ push: pushMock, replace: vi.fn() }),
}));

function jsonResponse(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } });
}

function renderLoginPage() {
  return render(
    <AuthProvider>
      <LoginPage />
    </AuthProvider>
  );
}

describe("LoginPage", () => {
  beforeEach(() => {
    pushMock.mockClear();
    // AuthProvider hydrates on mount by attempting a silent refresh -- for
    // these tests we simulate a logged-out visitor (refresh fails), which
    // is the realistic starting state for someone landing on /login.
    global.fetch = vi.fn().mockImplementation((url: string) => {
      if (String(url).includes("/auth/refresh")) {
        return Promise.resolve(jsonResponse(401, { success: false, error: { code: "UNAUTHORIZED", message: "no session" } }));
      }
      return Promise.resolve(jsonResponse(404, { success: false, error: { code: "NOT_FOUND", message: "unexpected call" } }));
    });
  });

  it("renders email and password fields", async () => {
    renderLoginPage();
    expect(await screen.findByLabelText("Email")).toBeInTheDocument();
    expect(screen.getByLabelText("Password")).toBeInTheDocument();
  });

  it("shows client-side validation errors without calling the API when the form is empty", async () => {
    const user = userEvent.setup();
    renderLoginPage();

    await screen.findByLabelText("Email");
    const fetchCallsBefore = (global.fetch as ReturnType<typeof vi.fn>).mock.calls.length;

    await user.click(screen.getByRole("button", { name: /log in/i }));

    expect(await screen.findByText("Email is required")).toBeInTheDocument();
    // no new network call should have been made for an invalid submission
    expect((global.fetch as ReturnType<typeof vi.fn>).mock.calls.length).toBe(fetchCallsBefore);
  });

  it("shows a validation error for a malformed email", async () => {
    const user = userEvent.setup();
    renderLoginPage();

    await user.type(await screen.findByLabelText("Email"), "not-an-email");
    await user.type(screen.getByLabelText("Password"), "somepassword");
    await user.click(screen.getByRole("button", { name: /log in/i }));

    expect(await screen.findByText("Enter a valid email address")).toBeInTheDocument();
  });

  it("submits valid credentials, calls the login API, and redirects on success", async () => {
    const user = userEvent.setup();
    global.fetch = vi.fn().mockImplementation((url: string, opts?: RequestInit) => {
      if (String(url).includes("/auth/refresh")) {
        return Promise.resolve(jsonResponse(401, { success: false, error: { code: "UNAUTHORIZED", message: "no session" } }));
      }
      if (String(url).includes("/auth/login")) {
        return Promise.resolve(
          jsonResponse(200, {
            success: true,
            data: {
              user: { id: 1, email: "jane@example.com", role: "candidate", is_active: true, email_verified: true },
              access_token: "tok-abc",
              expires_at: new Date().toISOString(),
            },
          })
        );
      }
      return Promise.resolve(jsonResponse(404, { success: false, error: { code: "NOT_FOUND", message: "unexpected" } }));
    });

    renderLoginPage();

    await user.type(await screen.findByLabelText("Email"), "jane@example.com");
    await user.type(screen.getByLabelText("Password"), "correct-password");
    await user.click(screen.getByRole("button", { name: /log in/i }));

    await waitFor(() => expect(pushMock).toHaveBeenCalledWith("/candidate/dashboard"));
  });

  it("redirects company accounts to the company dashboard", async () => {
    const user = userEvent.setup();
    global.fetch = vi.fn().mockImplementation((url: string) => {
      if (String(url).includes("/auth/refresh")) {
        return Promise.resolve(jsonResponse(401, { success: false, error: { code: "UNAUTHORIZED", message: "no session" } }));
      }
      if (String(url).includes("/auth/login")) {
        return Promise.resolve(
          jsonResponse(200, {
            success: true,
            data: {
              user: { id: 2, email: "company@example.com", role: "company", is_active: true, email_verified: true },
              access_token: "tok-company",
              expires_at: new Date().toISOString(),
            },
          })
        );
      }
      return Promise.resolve(jsonResponse(404, { success: false, error: { code: "NOT_FOUND", message: "unexpected" } }));
    });

    renderLoginPage();
    await user.type(await screen.findByLabelText("Email"), "company@example.com");
    await user.type(screen.getByLabelText("Password"), "correct-password");
    await user.click(screen.getByRole("button", { name: /log in/i }));

    await waitFor(() => expect(pushMock).toHaveBeenCalledWith("/company/dashboard"));
  });

  it("displays the API's error message when login fails (e.g. wrong password)", async () => {
    const user = userEvent.setup();
    global.fetch = vi.fn().mockImplementation((url: string) => {
      if (String(url).includes("/auth/refresh")) {
        return Promise.resolve(jsonResponse(401, { success: false, error: { code: "UNAUTHORIZED", message: "no session" } }));
      }
      if (String(url).includes("/auth/login")) {
        return Promise.resolve(
          jsonResponse(401, { success: false, error: { code: "UNAUTHORIZED", message: "invalid email or password" } })
        );
      }
      return Promise.resolve(jsonResponse(404, { success: false, error: { code: "NOT_FOUND", message: "unexpected" } }));
    });

    renderLoginPage();

    await user.type(await screen.findByLabelText("Email"), "jane@example.com");
    await user.type(screen.getByLabelText("Password"), "wrong-password");
    await user.click(screen.getByRole("button", { name: /log in/i }));

    expect(await screen.findByText("invalid email or password")).toBeInTheDocument();
    expect(pushMock).not.toHaveBeenCalled();
  });
});
