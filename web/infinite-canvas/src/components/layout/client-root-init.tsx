import { useEffect, type ReactNode } from "react";
import { App } from "antd";
import { useTranslation } from "react-i18next";

import { CanvasRefreshShell } from "@/components/canvas/canvas-refresh-shell";
import { usePromptSourceScheduler } from "@/hooks/use-prompt-source-scheduler";
import { loadPlatformModelChannels, type PlatformCanvasGroup } from "@/services/platform-models";
import { useConfigStore } from "@/stores/use-config-store";

const CANVAS_TOKEN_STORAGE_KEY = "new-api:canvas-dashboard-access-token";

type CanvasConfigPayload = {
    success: boolean;
    message?: string;
    data?: { groups?: PlatformCanvasGroup[] };
};

type CanvasConfigResult = { status: number; payload?: CanvasConfigPayload };

type RefreshPayload = { success: boolean; data?: { access_token?: string } };

function readCanvasToken() {
    try {
        return window.sessionStorage.getItem(CANVAS_TOKEN_STORAGE_KEY) || "";
    } catch {
        return "";
    }
}

function clearCanvasToken() {
    try {
        window.sessionStorage.removeItem(CANVAS_TOKEN_STORAGE_KEY);
    } catch {
        // Storage can be unavailable in locked-down browser contexts.
    }
}

async function refreshDashboardToken(signal: AbortSignal) {
    const response = await fetch("/api/user/auth/refresh", {
        method: "POST",
        credentials: "include",
        headers: { Accept: "application/json" },
        signal,
    });
    if (!response.ok) return "";
    const payload = (await response.json()) as RefreshPayload;
    const token = payload.data?.access_token?.trim() || "";
    if (!token) return "";
    try {
        window.sessionStorage.setItem(CANVAS_TOKEN_STORAGE_KEY, token);
    } catch {
        // The in-memory token is still usable for this page load.
    }
    return token;
}

async function requestCanvasConfig(token: string, signal: AbortSignal): Promise<CanvasConfigResult> {
    const response = await fetch("/api/canvas/config", {
        credentials: "include",
        headers: { Accept: "application/json", Authorization: `Bearer ${token}` },
        signal,
    });
    let payload: CanvasConfigPayload | undefined;
    try {
        payload = (await response.json()) as CanvasConfigPayload;
    } catch {
        payload = undefined;
    }
    return { status: response.status, payload };
}

export function ClientRootInit({ children }: { children: ReactNode }) {
    const { message } = App.useApp();
    const { t } = useTranslation();
    const applyPlatformChannels = useConfigStore((state) => state.applyPlatformChannels);
    const setPlatformModelError = useConfigStore((state) => state.setPlatformModelError);
    const setPlatformModelLoading = useConfigStore((state) => state.setPlatformModelLoading);
    const platformModelStatus = useConfigStore((state) => state.platformModelStatus);

    usePromptSourceScheduler();

    useEffect(() => {
        const controller = new AbortController();
        let disposed = false;
        const load = async () => {
            setPlatformModelLoading();
            let token = readCanvasToken();
            if (!token) token = await refreshDashboardToken(controller.signal);

            let result = token ? await requestCanvasConfig(token, controller.signal) : null;
            if (result?.status === 401) {
                clearCanvasToken();
                token = await refreshDashboardToken(controller.signal);
                result = token ? await requestCanvasConfig(token, controller.signal) : null;
            }
            if (!result || result.status === 401) {
                if (!disposed) window.location.replace(`/sign-in?redirect=${encodeURIComponent("/canvas")}`);
                return;
            }
            if (result.status < 200 || result.status >= 300 || !result.payload?.success || !result.payload.data) {
                throw new Error(result.payload?.message || t("apiErrors.requestFailed"));
            }
            const { channels, warning } = await loadPlatformModelChannels(result.payload.data.groups || [], controller.signal);
            if (!disposed) applyPlatformChannels(channels, warning);
        };

        void load().catch((error) => {
            if (error instanceof DOMException && error.name === "AbortError") return;
            if (!disposed) {
                const errorMessage = error instanceof Error ? error.message : t("apiErrors.requestFailed");
                setPlatformModelError(errorMessage);
                message.error(errorMessage);
            }
        });
        return () => {
            disposed = true;
            controller.abort();
        };
    }, [applyPlatformChannels, message, setPlatformModelError, setPlatformModelLoading, t]);

    return platformModelStatus === "loading" ? <CanvasRefreshShell /> : <>{children}</>;
}
