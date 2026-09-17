import axios from "axios";
import { nanoid } from "nanoid";

import i18n from "@/i18n";
import { dataUrlToFile } from "@/lib/image-utils";
import { uploadMediaFile, type UploadedFile } from "@/services/file-storage";
import { imageToDataUrl } from "@/services/image-storage";
import { boolConfig, buildApiUrl, modelOptionName, resolveModelChannel, resolveModelRequestConfig, resolveModelScript, type AiConfig, type ModelRequestConfig } from "@/stores/use-config-store";
import { runModelPlugin } from "./model-plugin";
import type { ReferenceImage } from "@/types/image";

type VideoResponse = { id: string; status?: string; error?: { message?: string }; url?: string; result_url?: string; video_url?: string; content?: { video_url?: string; url?: string } | null };
type ApiVideoResponse = VideoResponse | { code?: number | string; data?: VideoResponse | null; msg?: string; message?: string; error?: { message?: string } };
type ApiEnvelope<T> = T | { code?: number | string; data?: T | null; msg?: string; message?: string; error?: { message?: string } };
type RequestOptions = { signal?: AbortSignal };
const apiText = (key: string, options?: Record<string, unknown>) => i18n.t(`apiErrors.${key}`, options);
const miniMaxH3PollIntervalMs = 10_000;

export type VideoGenerationResult = { blob?: Blob; url?: string; mimeType?: string };
export type VideoGenerationTask = { id: string; provider: "openai" | "plugin"; model: string; videoAccountTokenId?: string };
export type VideoGenerationTaskState = { status: "pending" } | { status: "completed"; result: VideoGenerationResult } | { status: "failed"; error: string };

/** Results for scripted (plugin) video models, which run their own create+poll in one shot at task creation. */
const pluginVideoResults = new Map<string, VideoGenerationResult>();

function aiApiUrl(config: AiConfig, path: string) {
    return buildApiUrl(config.baseUrl, path);
}

function aiHeaders(config: AiConfig, contentType?: string, videoAccountTokenId?: string) {
    return {
        Authorization: `Bearer ${config.apiKey}`,
        ...(contentType ? { "Content-Type": contentType } : {}),
        ...(videoAccountTokenId ? { "X-Video-Creation-Token-Id": videoAccountTokenId } : {}),
    };
}

export async function requestVideoGeneration(config: AiConfig, prompt: string, references: ReferenceImage[] = [], options?: RequestOptions): Promise<VideoGenerationResult> {
    const task = await createVideoGenerationTask(config, prompt, references, options);
    for (let attempt = 0; attempt < 120; attempt += 1) {
        if (options?.signal?.aborted) throw new DOMException("Aborted", "AbortError");
        const state = await pollVideoGenerationTask(config, task, options);
        if (state.status === "completed") return state.result;
        if (state.status === "failed") throw new Error(state.error);
        if (attempt === 119) throw new Error(apiText("videoTimeout", { provider: "" }));
        await delay(isMiniMaxH3Model(task.model) || isStableVideoModel(task.model) ? miniMaxH3PollIntervalMs : 2500, options?.signal);
    }
    throw new Error(apiText("videoTimeout", { provider: "" }));
}

export async function createVideoGenerationTask(config: AiConfig, prompt: string, references: ReferenceImage[] = [], options?: RequestOptions): Promise<VideoGenerationTask> {
    const selectedModel = (config.model || config.videoModel).trim();
    const requestConfig = resolveModelRequestConfig(config, selectedModel);
    const script = resolveModelScript(config, selectedModel);
    if (script) return createPluginVideoTask(requestConfig, selectedModel, script, prompt, references, options);
    assertVideoConfig(requestConfig, requestConfig.model);
    return createOpenAIVideoTask(requestConfig, selectedModel, prompt, references, options);
}

export async function pollVideoGenerationTask(config: AiConfig, task: VideoGenerationTask, options?: RequestOptions): Promise<VideoGenerationTaskState> {
    if (task.provider === "plugin") {
        const result = pluginVideoResults.get(task.id);
        return result ? { status: "completed", result } : { status: "failed", error: apiText("pluginVideoExpired") };
    }
    const requestConfig = resolveModelRequestConfig(config, task.model);
    assertVideoConfig(requestConfig, requestConfig.model);
    return pollOpenAIVideoTask(requestConfig, task, options);
}

async function createPluginVideoTask(config: AiConfig, model: string, script: string, prompt: string, references: ReferenceImage[], options?: RequestOptions): Promise<VideoGenerationTask> {
    if (!config.baseUrl.trim()) throw new Error(apiText("baseUrlRequired"));
    if (!config.apiKey.trim()) throw new Error(apiText("apiKeyRequired"));
    const refs = await Promise.all(references.map((image) => imageToDataUrl(image)));
    const result = videoPluginResult(
        await runModelPlugin({
            capability: "video",
            script,
            config,
            prompt,
            images: refs,
            params: {
                seconds: normalizeVideoSeconds(config.videoSeconds),
                size: normalizeVideoSize(config.size),
                resolution: normalizeVideoResolution(config.vquality),
                ratio: config.size,
                generateAudio: boolConfig(config.videoGenerateAudio, true),
                watermark: boolConfig(config.videoWatermark, false),
            },
            signal: options?.signal,
        }),
    );
    const id = nanoid();
    pluginVideoResults.set(id, result);
    return { id, provider: "plugin", model };
}

function videoPluginResult(result: unknown): VideoGenerationResult {
    if (result instanceof Blob) return { blob: result };
    if (typeof result === "string") return { url: result, mimeType: "video/mp4" };
    if (result && typeof result === "object") {
        const record = result as Record<string, unknown>;
        if (record.blob instanceof Blob) return { blob: record.blob };
        const url = [record.url, record.video_url, record.result_url].find((value) => typeof value === "string" && value) as string | undefined;
        if (url) return { url, mimeType: "video/mp4" };
    }
    throw new Error(apiText("scriptNoVideo"));
}

export async function storeGeneratedVideo(result: VideoGenerationResult): Promise<UploadedFile> {
    if (result.blob) return uploadMediaFile(result.blob, "video");
    if (result.url) {
        try {
            return await uploadMediaFile(result.url, "video");
        } catch {
            return { url: result.url, storageKey: "", bytes: 0, mimeType: result.mimeType || "video/mp4" };
        }
    }
    throw new Error(apiText("noPlayableVideo"));
}

async function createOpenAIVideoTask(config: ModelRequestConfig, model: string, prompt: string, references: ReferenceImage[], options?: RequestOptions): Promise<VideoGenerationTask> {
    const modelName = modelOptionName(model);
    // The account selector is resolved per model, so a key that exposes models
    // from several accounts routes each one to its own upstream credential.
    const videoAccountTokenId = config.videoAccountTokenId?.trim() || "";
    if (videoAccountTokenId) {
        const task = await createVideoAccountTask(config, model, modelName, prompt, references, videoAccountTokenId, options);
        return { ...task, videoAccountTokenId };
    }
    if (isStableVideoModel(modelName)) return createStableVideoTask(config, model, modelName, prompt, references, options);

    const body = new FormData();
    const isMiniMaxH3 = isMiniMaxH3Model(model);
    const videoSize = normalizeVideoSize(config.size);
    body.append("model", modelName);
    body.append("prompt", prompt);
    body.append("seconds", isMiniMaxH3 ? normalizeMiniMaxH3Seconds(config.videoSeconds) : normalizeVideoSeconds(config.videoSeconds));
    if (isMiniMaxH3) {
        body.append("aspect_ratio", normalizeMiniMaxH3AspectRatio(config.size));
    } else {
        if (videoSize) body.append("size", videoSize);
        body.append("resolution_name", normalizeVideoResolution(config.vquality));
        body.append("preset", "normal");
    }
    const files = await Promise.all(references.slice(0, 7).map(async (image) => dataUrlToFile({ ...image, dataUrl: await imageToDataUrl(image) })));
    files.forEach((file) => body.append(isMiniMaxH3 ? "images" : "input_reference[]", file));
    try {
        const created = unwrapVideoResponse((await axios.post<ApiVideoResponse>(aiApiUrl(config, "/videos"), body, { headers: aiHeaders(config), signal: options?.signal })).data);
        if (!created.id) throw new Error(apiText("noVideoTaskId"));
        return { id: created.id, provider: "openai", model };
    } catch (error) {
        throw new Error(readAxiosError(error, apiText("videoTaskCreateFailed")));
    }
}

async function createVideoAccountTask(config: AiConfig, model: string, modelName: string, prompt: string, references: ReferenceImage[], tokenId: string, options?: RequestOptions): Promise<VideoGenerationTask> {
    const channel = resolveModelChannel(config, model);
    const metadata = channel.models.find((item) => item.name === modelName)?.video;
    try {
        const images = await Promise.all(
            references.slice(0, metadata?.maxImages && metadata.maxImages > 0 ? metadata.maxImages : 10).map(async (image) => {
                const file = await dataUrlToFile({ ...image, dataUrl: await imageToDataUrl(image) });
                const form = new FormData();
                form.append("type", "images");
                form.append("file", file);
                const response = await axios.post<{ images?: string[]; url?: string }>(`${window.location.origin}/api/sd-media/upload`, form, {
                    headers: aiHeaders(config, undefined, tokenId),
                    signal: options?.signal,
                    timeout: 90_000,
                });
                const url = response.data?.images?.[0]?.trim() || response.data?.url?.trim() || "";
                if (!url) throw new Error(apiText("referenceImageReadFailed"));
                return url;
            }),
        );
        const seconds = normalizeCatalogSeconds(config.videoSeconds, metadata?.durationsSeconds);
        const ratio = normalizeCatalogRatio(config.size, metadata?.ratios);
        const size = normalizeCatalogSize(config.size, metadata?.sizes, ratio);
        const body = {
            model: modelName,
            prompt,
            seconds,
            ...(ratio ? { aspect_ratio: ratio } : {}),
            ...(size ? { size } : {}),
            ...(images.length ? { images } : {}),
        };
        const created = unwrapVideoResponse(
            (
                await axios.post<ApiVideoResponse>(aiApiUrl(config, "/videos"), body, {
                    headers: aiHeaders(config, "application/json", tokenId),
                    signal: options?.signal,
                    timeout: 90_000,
                })
            ).data,
        );
        if (!created.id) throw new Error(apiText("noVideoTaskId"));
        return { id: created.id, provider: "openai", model };
    } catch (error) {
        throw new Error(readAxiosError(error, apiText("videoTaskCreateFailed")));
    }
}

async function createStableVideoTask(config: AiConfig, model: string, modelName: string, prompt: string, references: ReferenceImage[], options?: RequestOptions): Promise<VideoGenerationTask> {
    try {
        const images = await Promise.all(
            references.slice(0, stableVideoImageLimit(modelName)).map(async (image) => {
                const file = await dataUrlToFile({ ...image, dataUrl: await imageToDataUrl(image) });
                const upload = new FormData();
                upload.append("model", modelName);
                upload.append("type", "images");
                upload.append("file", file);
                const response = await axios.post<{ url?: string }>(aiApiUrl(config, "/videos/media"), upload, {
                    headers: aiHeaders(config),
                    signal: options?.signal,
                    timeout: 90_000,
                });
                const url = response.data?.url?.trim() || "";
                if (!url) throw new Error(apiText("referenceImageReadFailed"));
                return url;
            }),
        );
        const created = unwrapVideoResponse(
            (
                await axios.post<ApiVideoResponse>(
                    aiApiUrl(config, "/videos"),
                    {
                        model: modelName,
                        prompt,
                        seconds: normalizeStableVideoSeconds(modelName, config.videoSeconds),
                        aspect_ratio: normalizeStableVideoAspectRatio(modelName, config.size),
                        ...(images.length ? { images } : {}),
                    },
                    { headers: aiHeaders(config, "application/json"), signal: options?.signal, timeout: 90_000 },
                )
            ).data,
        );
        if (!created.id) throw new Error(apiText("noVideoTaskId"));
        return { id: created.id, provider: "openai", model };
    } catch (error) {
        throw new Error(readAxiosError(error, apiText("videoTaskCreateFailed")));
    }
}

async function pollOpenAIVideoTask(config: AiConfig, task: VideoGenerationTask, options?: RequestOptions): Promise<VideoGenerationTaskState> {
    try {
        // The selector is carried by the task: polling must use the account that
        // created the job, even if the user switched models in the meantime.
        const tokenId = task.videoAccountTokenId;
        const video = unwrapVideoResponse((await axios.get<ApiVideoResponse>(aiApiUrl(config, `/videos/${task.id}`), { headers: aiHeaders(config, undefined, tokenId), signal: options?.signal })).data);
        const url = videoResultUrl(video);
        if (url) return { status: "completed", result: await videoResultFromUrl(url, options) };
        if (video.status === "completed") {
            const content = await axios.get<Blob>(aiApiUrl(config, `/videos/${task.id}/content`), { headers: aiHeaders(config, undefined, tokenId), responseType: "blob", signal: options?.signal });
            await assertVideoBlob(content.data);
            return { status: "completed", result: { blob: content.data } };
        }
        if (video.status === "failed" || video.status === "cancelled") return { status: "failed", error: readApiErrorMessage(video.error?.message) || apiText("videoGenerationFailed") };
        return { status: "pending" };
    } catch (error) {
        throw new Error(readAxiosError(error, apiText("videoTaskQueryFailed")));
    }
}

async function videoResultFromUrl(url: string, options?: RequestOptions): Promise<VideoGenerationResult> {
    try {
        const response = await axios.get<Blob>(url, { responseType: "blob", signal: options?.signal });
        await assertVideoBlob(response.data);
        return { blob: response.data };
    } catch (error) {
        if (axios.isCancel(error) || options?.signal?.aborted) throw error;
        return { url, mimeType: "video/mp4" };
    }
}

function assertVideoConfig(config: AiConfig, model: string) {
    if (!model) throw new Error(apiText("videoModelRequired"));
    if (!config.baseUrl.trim()) throw new Error(apiText("baseUrlRequired"));
    if (!config.apiKey.trim()) throw new Error(apiText("apiKeyRequired"));
    if (config.apiFormat === "gemini") throw new Error(apiText("geminiVideoUnsupported"));
}

function normalizeVideoSeconds(value: string) {
    const seconds = Math.floor(Number(value) || 6);
    return String(Math.max(1, Math.min(20, seconds)));
}

function normalizeCatalogSeconds(value: string, supported?: number[]) {
    const requested = Math.floor(Number(value) || 4);
    const values = (supported || []).filter((item) => Number.isInteger(item) && item > 0 && item <= 3600);
    if (!values.length) return Math.max(1, Math.min(3600, requested));
    if (values.includes(requested)) return requested;
    return values.reduce((closest, item) => (Math.abs(item - requested) < Math.abs(closest - requested) ? item : closest), values[0]);
}

function normalizeMiniMaxH3Seconds(value: string) {
    const seconds = Math.floor(Number(value) || 6);
    return String(Math.max(4, Math.min(15, seconds)));
}

function isMiniMaxH3Model(model: string) {
    return modelOptionName(model).startsWith("minimax-h3-");
}

function isStableVideoModel(model: string) {
    return modelOptionName(model).startsWith("seedance2.");
}

function stableVideoImageLimit(model: string) {
    if (model.includes("fast-720p") || model.includes("standard-720p")) return 4;
    if (model.startsWith("seedance2.5-")) return 30;
    return 9;
}

function normalizeStableVideoSeconds(model: string, value: string) {
    const seconds = Math.floor(Number(value) || 6);
    if (model.includes("fast-720p") || model.includes("standard-720p")) return seconds <= 10 ? 10 : 15;
    if (model.startsWith("seedance2.5-")) return Math.max(4, Math.min(30, seconds));
    if (model.includes("facecheck-720p") || model.includes("facefree-720p")) return Math.max(5, Math.min(15, seconds));
    return Math.max(4, Math.min(15, seconds));
}

function normalizeStableVideoAspectRatio(model: string, value: string) {
    if (model.includes("facecheck-720p") || model.includes("facefree-720p")) return "9:16";
    const ratio = normalizeMiniMaxH3AspectRatio(value);
    let supported = ["16:9", "9:16"];
    if (model.includes("stable-900-720p")) supported = ["16:9", "9:16", "21:9"];
    if (model.includes("full-480p") || model.includes("seedance2.0-stable-full-720p")) supported = ["16:9", "9:16", "1:1", "4:3", "3:4", "21:9"];
    if (model.includes("seedance2.5-stable-480p")) supported = ["16:9", "9:16", "1:1"];
    return supported.includes(ratio) ? ratio : "16:9";
}

function normalizeMiniMaxH3AspectRatio(value: string) {
    const ratio = value.trim();
    if (["16:9", "9:16", "1:1", "2:3", "3:2", "3:4", "4:3", "21:9"].includes(ratio)) return ratio;
    const [width, height] = ratio.split("x").map(Number);
    if (Number.isFinite(width) && Number.isFinite(height) && width > 0 && height > 0) {
        if (width === height) return "1:1";
        if (width > height) return width / height > 2 ? "21:9" : "16:9";
        return "9:16";
    }
    return "16:9";
}

function normalizeCatalogRatio(value: string, supported?: string[]) {
    const ratio = normalizeMiniMaxH3AspectRatio(value);
    if (!supported?.length) return ratio;
    return supported.includes(ratio) ? ratio : supported[0];
}

function normalizeCatalogSize(value: string, supported: string[] | undefined, ratio: string) {
    if (!supported?.length) return "";
    if (supported.includes(value)) return value;
    const ratioParts = ratio.split(":").map(Number);
    const target = ratioParts.length === 2 && ratioParts[1] ? ratioParts[0] / ratioParts[1] : 0;
    if (!target) return supported[0];
    return supported.reduce((best, candidate) => {
        const parts = candidate.match(/^(\d+)x(\d+)$/i);
        const current = parts ? Number(parts[1]) / Number(parts[2]) : 0;
        const previous = best.match(/^(\d+)x(\d+)$/i);
        const previousRatio = previous ? Number(previous[1]) / Number(previous[2]) : 0;
        return Math.abs(current - target) < Math.abs(previousRatio - target) ? candidate : best;
    }, supported[0]);
}

function normalizeVideoSize(value: string) {
    if (value === "auto") return null;
    const size = value || "1280x720";
    if (/^\d+x\d+$/.test(size)) return size;
    return ["9:16", "2:3", "3:4"].includes(size) ? "720x1280" : "1280x720";
}

function normalizeVideoResolution(value: string) {
    if (value === "low") return "480p";
    if (value === "auto" || value === "high" || value === "medium") return "720p";
    const resolution = value.replace(/p$/i, "") || "720";
    return `${resolution}p`;
}

function unwrapVideoResponse(payload: ApiVideoResponse) {
    return unwrapEnvelope(payload, apiText("noVideoTask"));
}

function unwrapEnvelope<T>(payload: ApiEnvelope<T>, emptyMessage: string): T {
    if (!payload) throw new Error(emptyMessage);
    if (typeof payload === "object" && "code" in payload && payload.code !== undefined) {
        if (payload.code !== 0 && payload.code !== "0") throw new Error(readApiErrorMessage(payload) || apiText("requestFailed"));
        if (!payload.data) throw new Error(emptyMessage);
        return payload.data;
    }
    return payload as T;
}

function videoResultUrl(payload: VideoResponse) {
    return [payload.video_url, payload.result_url, payload.url, payload.content?.video_url, payload.content?.url].find((url) => typeof url === "string" && (isPublicMediaUrl(url) || /\.mp4(\?|#|$)/i.test(url)));
}

function readApiErrorMessage(value: unknown): string {
    if (!value) return "";
    if (typeof value === "string") {
        try {
            const parsed = JSON.parse(value);
            const inner = readApiErrorMessage(parsed) || value;
            if (inner === value && typeof parsed === "object" && Object.keys(parsed).length === 0) return "";
            return inner;
        } catch {
            if (/<[a-z][\s\S]*>/i.test(value)) return apiText("htmlError", { preview: `${value.slice(0, 80)}...` });
            return value;
        }
    }
    if (typeof value !== "object") return "";
    const payload = value as { msg?: unknown; message?: unknown; error?: unknown; detail?: unknown };
    // error may be a string or an object containing a message.
    const errorMsg = typeof payload.error === "string" ? payload.error : (payload.error as { message?: unknown })?.message;
    return readApiErrorMessage(payload.msg) || readApiErrorMessage(payload.message) || readApiErrorMessage(errorMsg) || readApiErrorMessage(payload.detail) || "";
}

function readAxiosError(error: unknown, fallback: string) {
    if (axios.isCancel(error)) return apiText("requestCanceled");
    if (axios.isAxiosError<{ error?: { message?: string }; msg?: string; message?: string; code?: number | string }>(error)) {
        if (!error.response && error.code === "ERR_NETWORK") return apiText("requestFailed");
        const responseData = error.response?.data;
        return readApiErrorMessage(responseData) || statusMessage(error.response?.status, fallback);
    }
    if (error instanceof DOMException && error.name === "AbortError") return apiText("requestCanceled");
    return error instanceof Error ? readApiErrorMessage(error.message) || error.message : fallback;
}

function statusMessage(status: number | undefined, fallback: string) {
    if (status === 401 || status === 403) return apiText("authenticationFailed");
    if (status === 429) return apiText("rateLimited");
    return status ? `${fallback}（${status}）` : fallback;
}

async function assertVideoBlob(blob: Blob) {
    if (!blob.type.includes("json")) return;
    let payload: { code?: number; msg?: string; error?: { message?: string } };
    try {
        payload = JSON.parse(await blob.text()) as { code?: number; msg?: string; error?: { message?: string } };
    } catch {
        return;
    }
    if (typeof payload.code === "number" && payload.code !== 0) throw new Error(readApiErrorMessage(payload) || apiText("videoDownloadFailed"));
    if (payload.error?.message) throw new Error(readApiErrorMessage(payload.error.message) || payload.error.message);
}

function isPublicMediaUrl(value: string) {
    return /^https?:\/\//i.test(value || "");
}

function delay(ms: number, signal?: AbortSignal) {
    return new Promise<void>((resolve, reject) => {
        if (signal?.aborted) {
            reject(new DOMException("Aborted", "AbortError"));
            return;
        }
        const timer = setTimeout(resolve, ms);
        signal?.addEventListener(
            "abort",
            () => {
                clearTimeout(timer);
                reject(new DOMException("Aborted", "AbortError"));
            },
            { once: true },
        );
    });
}
