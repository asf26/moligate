/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import type { AnchorHTMLAttributes } from "react";
import { afterEach, describe, expect, test, vi } from "vitest";

import { getApiKeys } from "@/features/keys/api";
import { useAuthStore, type AuthBundle } from "@/stores/auth-store";

import { CanvasWorkspace } from "../-canvas-workspace";

vi.mock("@tanstack/react-router", () => ({
  Link: ({
    to,
    search,
    ...props
  }: {
    to: string;
    search?: Record<string, boolean | string | undefined>;
  } & AnchorHTMLAttributes<HTMLAnchorElement>) => {
    const query = search
      ? new URLSearchParams(
          Object.entries(search).reduce<Record<string, string>>(
            (params, [key, value]) => {
              if (value !== undefined) params[key] = String(value);
              return params;
            },
            {},
          ),
        ).toString()
      : "";

    return <a href={`${to}${query ? `?${query}` : ""}`} {...props} />;
  },
}));

vi.mock("@/features/keys/api", () => ({
  getApiKeys: vi.fn(),
}));

const mockedGetApiKeys = vi.mocked(getApiKeys);

const authBundle: AuthBundle = {
  access_token: "canvas-dashboard-token",
  token_type: "Bearer",
  access_expires_at: 2_000_000_000,
  user: { id: 1, username: "canvas-user", role: 1 },
  session: {
    sid: "canvas-session",
    current: true,
    login_method: "password",
    ip: "127.0.0.1",
    user_agent: "vitest",
    created_at: 1,
    last_active_at: 1,
    expires_at: 2_000_000_000,
  },
};

afterEach(() => {
  window.sessionStorage.clear();
  useAuthStore.getState().auth.reset("idle");
  mockedGetApiKeys.mockReset();
});

function renderWorkspace(configMode = false) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });

  return render(
    <QueryClientProvider client={queryClient}>
      <CanvasWorkspace configMode={configMode} />
    </QueryClientProvider>,
  );
}

describe("CanvasWorkspace", () => {
  test("hands the dashboard token to the latest unified canvas editor without replacing it", async () => {
    mockedGetApiKeys.mockResolvedValue({
      success: true,
      data: {
        items: [
          {
            id: 1,
            name: "image-key",
            key: "sk-image",
            status: 1,
            remain_quota: 1,
            used_quota: 0,
            unlimited_quota: true,
            expired_time: -1,
            created_time: 1,
            accessed_time: 1,
            group: "",
            auto_groups: null,
            cross_group_retry: false,
            model_limits_enabled: true,
            model_limits: "gpt-image-2",
            allow_ips: "",
          },
        ],
        total: 1,
        page: 1,
        page_size: 100,
      },
    });
    useAuthStore.getState().auth.setBundle(authBundle);

    renderWorkspace();

    await waitFor(() => {
      expect(
        window.sessionStorage.getItem("new-api:canvas-dashboard-access-token"),
      ).toBe(authBundle.access_token);
    });

    const iframe = await screen.findByTitle("Creative Canvas");
    expect(iframe).toHaveAttribute("src", "/canvas-app/?mode=recent");
    expect(screen.getByText("Loading...")).toBeInTheDocument();
    fireEvent.load(iframe);
    await waitFor(() =>
      expect(screen.queryByText("Loading...")).not.toBeInTheDocument(),
    );
    expect(
      screen.getByRole("heading", { name: "AI Creation" }),
    ).toBeInTheDocument();

    expect(screen.getByTitle("Creative Canvas")).toBe(iframe);
    expect(iframe).toHaveAttribute("src", "/canvas-app/?mode=recent");
  });

  test("guides users to create a gpt-image-2 key before opening the workspace", async () => {
    mockedGetApiKeys.mockResolvedValue({
      success: true,
      data: { items: [], total: 0, page: 1, page_size: 100 },
    });
    useAuthStore.getState().auth.setBundle(authBundle);

    renderWorkspace();

    expect(
      await screen.findByText(
        "Create an API key to use AI Creation with gpt-image-2.",
      ),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("link", { name: "Create API Key" }),
    ).toHaveAttribute("href", "/keys");
    expect(screen.getByRole("link", { name: "Configuration" })).toHaveAttribute(
      "href",
      "/canvas?standalone=true&config=true",
    );
    expect(screen.queryByTitle("Creative Canvas")).not.toBeInTheDocument();
  });

  test("opens the platform-managed configuration page without an image key", async () => {
    useAuthStore.getState().auth.setBundle(authBundle);

    renderWorkspace(true);

    const iframe = await screen.findByTitle("Creative Canvas");
    expect(iframe).toHaveAttribute("src", "/canvas-app/config?mode=recent");
    expect(mockedGetApiKeys).not.toHaveBeenCalled();
  });
});
