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
import { useQuery } from "@tanstack/react-query";
import { ArrowRight, ImagePlus, KeyRound, LoaderCircle } from "lucide-react";
import { Link } from "@tanstack/react-router";
import { useEffect, useRef, useState, type ReactNode } from "react";
import { useTranslation } from "react-i18next";

import { API_KEY_STATUS } from "@/features/keys/constants";
import { getApiKeys } from "@/features/keys/api";
import type { ApiKey } from "@/features/keys/types";
import { useAuthStore } from "@/stores/auth-store";

/* eslint-disable react/iframe-missing-sandbox -- the same-origin child needs scripts and storage. */

const CANVAS_TOKEN_STORAGE_KEY = "new-api:canvas-dashboard-access-token";
const CANVAS_APP_BASE_PATH = "/canvas-app/";
const INTERNAL_CANVAS_KEY_PREFIX = "infinite-canvas:";
/**
 * Keep the latest infinite-canvas app inside the authenticated shell. The
 * dashboard mounts one workspace; its internal navigation owns the canvas,
 * image, and video workbenches without replacing the outer dashboard page.
 * The child app reads this handoff token before requesting its scoped gateway
 * configuration.
 */
export function CanvasWorkspace({
  configMode = false,
}: { configMode?: boolean } = {}) {
  const { t } = useTranslation();
  const accessToken = useAuthStore((state) => state.auth.accessToken);
  const userId = useAuthStore((state) => state.auth.user?.id);
  const [handoffToken, setHandoffToken] = useState<string | null>(null);
  const [iframeLoaded, setIframeLoaded] = useState(false);
  const iframeRef = useRef<HTMLIFrameElement>(null);

  const apiKeysQuery = useQuery({
    queryKey: ["canvas-image-generation-key", userId],
    queryFn: () => getApiKeys({ p: 1, size: 100 }),
    enabled: Boolean(accessToken && userId && !configMode),
    retry: false,
    staleTime: 0,
    refetchOnMount: "always",
  });

  const source = configMode
    ? `${CANVAS_APP_BASE_PATH}config?mode=recent`
    : `${CANVAS_APP_BASE_PATH}?mode=recent`;

  const apiKeys = apiKeysQuery.data?.data?.items ?? [];
  const imageGenerationKeyReady = hasGptImage2Access(apiKeys);
  const shouldPromptForImageKey = Boolean(
    !configMode &&
    handoffToken &&
    !apiKeysQuery.isLoading &&
    !apiKeysQuery.isError &&
    apiKeysQuery.data?.success &&
    !imageGenerationKeyReady,
  );
  const canRenderCanvas = Boolean(
    handoffToken &&
    !shouldPromptForImageKey &&
    (configMode ||
      (!apiKeysQuery.isLoading &&
        (apiKeysQuery.isError ||
          !apiKeysQuery.data ||
          !apiKeysQuery.data.success ||
          imageGenerationKeyReady))),
  );

  useEffect(() => {
    if (!accessToken) {
      // eslint-disable-next-line react-hooks/set-state-in-effect
      setHandoffToken(null);
      return;
    }

    try {
      window.sessionStorage.setItem(CANVAS_TOKEN_STORAGE_KEY, accessToken);
    } catch {
      // The child app can fall back to the refresh cookie if storage is blocked.
    }

    // Render the iframe only after the token handoff is complete.
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setHandoffToken(accessToken);
  }, [accessToken]);

  useEffect(() => {
    // A fresh handoff starts a fresh iframe load. Keep the loading surface
    // visible until the embedded editor has painted its first document.
    setIframeLoaded(false);
  }, [handoffToken]);

  let workspaceContent: ReactNode = (
    <CanvasLoadingState label={t("Loading...")} />
  );
  if (shouldPromptForImageKey) {
    workspaceContent = <ImageKeyRequiredState />;
  } else if (canRenderCanvas) {
    workspaceContent = (
      <div className="relative size-full" aria-busy={!iframeLoaded}>
        {!iframeLoaded ? (
          <div className="absolute inset-0 z-10">
            <CanvasLoadingState label={t("Loading...")} />
          </div>
        ) : null}
        <iframe
          ref={iframeRef}
          title={t("Creative Canvas")}
          src={source}
          onLoad={() => setIframeLoaded(true)}
          className={`block size-full border-0 transition-opacity duration-300 ${iframeLoaded ? "opacity-100" : "opacity-0"}`}
          aria-label={t("Creative Canvas")}
          sandbox="allow-downloads allow-forms allow-modals allow-popups allow-same-origin allow-scripts allow-top-navigation-by-user-activation"
        />
      </div>
    );
  }

  return (
    <div
      data-testid="ai-creation-workspace"
      className="bg-background relative h-full min-h-0 min-w-0"
    >
      <h1 className="sr-only">{t("AI Creation")}</h1>
      {workspaceContent}
    </div>
  );
}

export function hasGptImage2Access(apiKeys: ApiKey[]) {
  return apiKeys.some((apiKey) => {
    if (apiKey.status !== API_KEY_STATUS.ENABLED) return false;
    if (apiKey.name.trim().toLowerCase().startsWith(INTERNAL_CANVAS_KEY_PREFIX))
      return false;
    if (!apiKey.model_limits_enabled) return true;

    return (apiKey.model_limits || "")
      .split(/[\s,]+/)
      .map((model) => model.trim().toLowerCase())
      .some((model) => model === "gpt-image-2");
  });
}

function CanvasLoadingState({ label }: { label: string }) {
  return (
    <main
      className="relative grid h-full min-h-0 place-items-center overflow-hidden bg-[#f8fbff] px-6 text-slate-700"
      aria-live="polite"
    >
      <div className="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_50%_40%,rgba(59,130,246,0.13),transparent_42%),linear-gradient(135deg,rgba(239,246,255,0.9),rgba(248,250,252,0.96))]" />
      <section className="relative flex w-full max-w-sm flex-col items-center rounded-3xl border border-blue-100/90 bg-white/85 px-8 py-9 text-center shadow-[0_24px_70px_-36px_rgba(37,99,235,0.55)] backdrop-blur-xl">
        <div className="relative mb-5 grid size-16 place-items-center rounded-2xl border border-blue-100 bg-blue-50 text-blue-600 shadow-inner">
          <span className="absolute inset-0 animate-ping rounded-2xl border border-blue-300/60" />
          <LoaderCircle
            aria-hidden="true"
            className="relative size-7 animate-spin"
          />
        </div>
        <p className="text-sm font-semibold tracking-wide text-slate-800">
          {label}
        </p>
        <div className="mt-6 h-1.5 w-44 overflow-hidden rounded-full bg-blue-50">
          <div className="h-full w-1/2 animate-[canvas-loading_1.6s_ease-in-out_infinite] rounded-full bg-gradient-to-r from-blue-500 to-cyan-400" />
        </div>
      </section>
      <style>{`@keyframes canvas-loading { 0%, 100% { transform: translateX(-100%); } 50% { transform: translateX(200%); } }`}</style>
    </main>
  );
}

function ImageKeyRequiredState() {
  const { t } = useTranslation();

  return (
    <main className="relative grid h-full min-h-0 place-items-center overflow-hidden bg-[#f8fbff] px-6 py-10 text-slate-900">
      <div className="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_20%_20%,rgba(96,165,250,0.2),transparent_33%),radial-gradient(circle_at_85%_80%,rgba(45,212,191,0.14),transparent_30%)]" />
      <section className="relative w-full max-w-lg rounded-[28px] border border-blue-100 bg-white/90 p-8 shadow-[0_30px_90px_-44px_rgba(37,99,235,0.6)] backdrop-blur-xl sm:p-10">
        <div className="flex size-14 items-center justify-center rounded-2xl border border-blue-100 bg-gradient-to-br from-blue-50 to-cyan-50 text-blue-600">
          <ImagePlus aria-hidden="true" className="size-6" />
        </div>
        <p className="mt-7 text-xs font-semibold uppercase tracking-[0.18em] text-blue-600">
          {t("Image generation")}
        </p>
        <h2 className="mt-2 text-2xl font-semibold tracking-tight text-slate-950">
          {t("Create API Key")}
        </h2>
        <p className="mt-3 text-sm leading-6 text-slate-600">
          {t("Create an API key to use AI Creation with gpt-image-2.")}
        </p>
        <div className="mt-6 flex items-center gap-3 rounded-2xl border border-blue-100 bg-blue-50/70 px-4 py-3 text-sm text-slate-700">
          <KeyRound
            aria-hidden="true"
            className="size-4 shrink-0 text-blue-600"
          />
          <span>{t("API Key")}</span>
          <code className="ml-auto rounded-md bg-white px-2 py-1 font-mono text-xs text-blue-700 shadow-sm">
            gpt-image-2
          </code>
        </div>
        <Link
          to="/keys"
          className="mt-7 inline-flex h-11 w-full items-center justify-center gap-2 rounded-xl bg-gradient-to-r from-blue-600 to-cyan-500 px-5 text-sm font-semibold text-white shadow-[0_12px_24px_-12px_rgba(37,99,235,0.9)] transition hover:-translate-y-0.5 hover:shadow-[0_16px_28px_-12px_rgba(37,99,235,0.95)]"
        >
          {t("Create API Key")}
          <ArrowRight aria-hidden="true" className="size-4" />
        </Link>
        <Link
          to="/canvas"
          search={{ standalone: true, config: true }}
          className="mt-3 inline-flex h-10 w-full items-center justify-center rounded-xl border border-blue-200 bg-white px-5 text-sm font-semibold text-blue-700 transition hover:-translate-y-0.5 hover:border-blue-300 hover:bg-blue-50"
        >
          {t("Configuration")}
        </Link>
      </section>
    </main>
  );
}
