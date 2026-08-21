// @vitest-environment happy-dom

import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { DataApiError } from "@/lib/data-api-error";
import { AuthSessionPersistenceError } from "@/model/auth";

import { LoginForm } from "./login-form";
import { useLoginController } from "./use-login-controller";

const mocks = vi.hoisted(() => ({
  login: vi.fn(),
  navigate: vi.fn(),
  resolveAuthRedirect: vi.fn(),
  saveAuthSession: vi.fn(),
}));

const translations: Record<string, string> = {
  hidePassword: "Hide password",
  invalidCredentials: "The username or password is incorrect.",
  password: "Password",
  persistenceError: "Your session could not be saved.",
  showPassword: "Show password",
  submit: "Log in",
  submitting: "Logging in...",
  unavailable: "The service is unavailable.",
  unknownError: "We couldn't log you in.",
  username: "Username",
};

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => translations[key] ?? key }),
}));

vi.mock("@tanstack/react-router", () => ({
  useNavigate: () => mocks.navigate,
}));

vi.mock("./resolve-auth-redirect", () => ({
  resolveAuthRedirect: mocks.resolveAuthRedirect,
}));

vi.mock("@/model/auth", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/model/auth")>()),
  authApi: { login: mocks.login },
  saveAuthSession: mocks.saveAuthSession,
}));

const loginResponse = {
  accessToken: "access-token",
  tokenType: "Bearer" as const,
  expiresIn: 3_600,
  user: {
    id: 7,
    username: "artist",
    email: "artist@example.com",
  },
};

beforeEach(() => {
  vi.clearAllMocks();
  mocks.login.mockResolvedValue(loginResponse);
  mocks.navigate.mockResolvedValue(undefined);
  mocks.resolveAuthRedirect.mockReturnValue("/projects/7");
});

afterEach(cleanup);

describe("login form", () => {
  it("enables submission only after both credentials are entered", async () => {
    const user = userEvent.setup();
    renderLogin();
    const submit = screen.getByRole("button", { name: "Log in" });

    expect((submit as HTMLButtonElement).disabled).toBe(true);
    await user.type(screen.getByLabelText("Username"), "artist");
    expect((submit as HTMLButtonElement).disabled).toBe(true);
    await user.type(passwordInput(), "secret");
    expect((submit as HTMLButtonElement).disabled).toBe(false);
  });

  it("toggles password visibility through its accessible control", async () => {
    const user = userEvent.setup();
    renderLogin();
    const password = passwordInput();

    expect(password.type).toBe("password");
    await user.click(screen.getByRole("button", { name: "Show password" }));
    expect(password.type).toBe("text");
    await user.click(screen.getByRole("button", { name: "Hide password" }));
    expect(password.type).toBe("password");
  });

  it("logs in, persists the session, and follows the safe redirect", async () => {
    const user = userEvent.setup();
    renderLogin("/projects/7");

    await submitCredentials(user, "  artist  ", " secret ");

    await waitFor(() =>
      expect(mocks.navigate).toHaveBeenCalledWith({
        href: "/projects/7",
        replace: true,
      }),
    );
    expect(mocks.login).toHaveBeenCalledWith({
      username: "artist",
      password: " secret ",
    });
    expect(mocks.saveAuthSession).toHaveBeenCalledWith(loginResponse);
    expect(mocks.resolveAuthRedirect).toHaveBeenCalledWith("/projects/7");
  });

  it.each([
    [
      new DataApiError("UNAUTHORIZED", "invalid credentials"),
      "The username or password is incorrect.",
    ],
    [
      new DataApiError("UNAVAILABLE", "service unavailable"),
      "The service is unavailable.",
    ],
    [new Error("unexpected"), "We couldn't log you in."],
  ])(
    "shows the mapped error and clears it when editing",
    async (error, message) => {
      const user = userEvent.setup();
      mocks.login.mockRejectedValueOnce(error);
      renderLogin();

      await submitCredentials(user, "artist", "secret");

      expect((await screen.findByRole("alert")).textContent).toContain(message);
      expect(mocks.saveAuthSession).not.toHaveBeenCalled();
      expect(mocks.navigate).not.toHaveBeenCalled();

      await user.type(screen.getByLabelText("Username"), "2");
      expect(screen.queryByRole("alert")).toBeNull();
    },
  );

  it("reports persistence failures without navigating", async () => {
    const user = userEvent.setup();
    mocks.saveAuthSession.mockImplementationOnce(() => {
      throw new AuthSessionPersistenceError();
    });
    renderLogin();

    await submitCredentials(user, "artist", "secret");

    expect((await screen.findByRole("alert")).textContent).toContain(
      "Your session could not be saved.",
    );
    expect(mocks.login).toHaveBeenCalledOnce();
    expect(mocks.navigate).not.toHaveBeenCalled();
  });
});

function LoginHarness({ redirectTo }: { redirectTo?: string }) {
  return <LoginForm login={useLoginController(redirectTo)} />;
}

function renderLogin(redirectTo?: string) {
  return render(<LoginHarness redirectTo={redirectTo} />);
}

async function submitCredentials(
  user: ReturnType<typeof userEvent.setup>,
  username: string,
  password: string,
) {
  await user.type(screen.getByLabelText("Username"), username);
  await user.type(passwordInput(), password);
  await user.click(screen.getByRole("button", { name: "Log in" }));
}

function passwordInput() {
  return screen.getByLabelText("Password", {
    selector: "input",
  }) as HTMLInputElement;
}
