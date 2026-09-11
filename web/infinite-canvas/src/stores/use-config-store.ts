import { useMemo } from "react";
import { create } from "zustand";
import { persist } from "zustand/middleware";
import { nanoid } from "nanoid";

import i18n from "@/i18n";

export type ApiCallFormat = "openai" | "gemini";
export type ModelCapability = "image" | "video" | "text" | "audio";
export type ReasoningEffort = "auto" | "low" | "medium" | "high" | "xhigh";

export type VideoModelMetadata = {
    group?: string;
    durationsSeconds?: number[];
    ratios?: string[];
    sizes?: string[];
    maxImages?: number;
    maxVideos?: number;
    maxAudios?: number;
    supportsFirstLastFrame?: boolean;
    pricingMode?: string;
};

export type ChannelModel = {
    name: string;
    capability: ModelCapability;
    script?: string;
    video?: VideoModelMetadata;
};

export type ModelChannel = {
    id: string;
    name: string;
    baseUrl: string;
    apiKey: string;
    apiFormat: ApiCallFormat;
    models: ChannelModel[];
    videoAccountTokenId?: string;
};

export type AiConfig = {
    channelMode: "remote" | "local";
    baseUrl: string;
    apiKey: string;
    apiFormat: ApiCallFormat;
    channels: ModelChannel[];
    /** The platform-managed group used by the default creation models. */
    platformChannelId: string;
    model: string;
    imageModel: string;
    videoModel: string;
    textModel: string;
    audioModel: string;
    audioVoice: string;
    audioFormat: string;
    audioSpeed: string;
    audioInstructions: string;
    videoSeconds: string;
    vquality: string;
    videoGenerateAudio: string;
    videoWatermark: string;
    systemPrompt: string;
    reasoningEffort: ReasoningEffort;
    models: string[];
    quality: string;
    size: string;
    background: string;
    count: string;
    canvasImageCount: string;
};

export type PlatformModelStatus = "loading" | "ready" | "empty" | "error";

export type WebdavSyncConfig = {
    url: string;
    username: string;
    password: string;
    directory: string;
    lastSyncedAt: string;
};
export type ConfigTabKey = "channels" | "preferences" | "prompt-sources" | "webdav" | "local-storage";

export const CONFIG_STORE_KEY = "infinite-canvas:ai_config_store";
const CHANNEL_MODEL_SEPARATOR = "::";
const OPENAI_BASE_URL = `${window.location.origin}/v1`;
const GEMINI_BASE_URL = "https://generativelanguage.googleapis.com";
const PLATFORM_MANAGED_CONFIG_KEYS = new Set<keyof AiConfig>(["channelMode", "baseUrl", "apiKey", "apiFormat", "channels", "platformChannelId", "models"]);

export const defaultConfig: AiConfig = {
    channelMode: "local",
    baseUrl: OPENAI_BASE_URL,
    apiKey: "",
    apiFormat: "openai",
    channels: [],
    platformChannelId: "",
    model: "",
    imageModel: "",
    videoModel: "",
    textModel: "",
    audioModel: "",
    audioVoice: "alloy",
    audioFormat: "mp3",
    audioSpeed: "1",
    audioInstructions: "",
    videoSeconds: "6",
    vquality: "720",
    videoGenerateAudio: "true",
    videoWatermark: "false",
    systemPrompt: "",
    reasoningEffort: "auto",
    models: [],
    quality: "auto",
    size: "1:1",
    background: "",
    count: "1",
    canvasImageCount: "3",
};

export const defaultWebdavSyncConfig: WebdavSyncConfig = {
    url: "",
    username: "",
    password: "",
    directory: "infinite-canvas",
    lastSyncedAt: "",
};

type ConfigStore = {
    config: AiConfig;
    webdav: WebdavSyncConfig;
    platformModelStatus: PlatformModelStatus;
    platformModelError: string;
    isConfigOpen: boolean;
    configTab: ConfigTabKey;
    shouldPromptContinue: boolean;
    updateConfig: <K extends keyof AiConfig>(key: K, value: AiConfig[K]) => void;
    setPlatformModelLoading: () => void;
    setPlatformModelError: (message: string) => void;
    applyPlatformChannels: (channels: ModelChannel[], warning?: string) => void;
    selectPlatformChannel: (channelId: string) => void;
    updateWebdavConfig: <K extends keyof WebdavSyncConfig>(key: K, value: WebdavSyncConfig[K]) => void;
    isAiConfigReady: (config: AiConfig, model: string) => boolean;
    openConfigDialog: (shouldPromptContinue?: boolean, tab?: ConfigTabKey) => void;
    setConfigDialogOpen: (isOpen: boolean) => void;
    clearPromptContinue: () => void;
};

const VIDEO_KEYWORDS = ["video", "sora", "seedance", "minimax-h3", "veo", "kling", "wan", "hailuo"];

export function boolConfig(value: string, fallback: boolean) {
    return value ? value === "true" : fallback;
}
const AUDIO_KEYWORDS = ["audio", "tts", "speech", "voice", "music", "sound"];
const IMAGE_KEYWORDS = ["seedream", "gpt-image", "image", "dall-e", "dalle", "imagen", "flux", "sdxl", "stable-diffusion", "midjourney"];

/** Best-effort default capability for a freshly fetched model name; user can override in the channel editor. */
export function guessCapability(name: string): ModelCapability {
    const value = name.toLowerCase();
    if (VIDEO_KEYWORDS.some((keyword) => value.includes(keyword))) return "video";
    if (AUDIO_KEYWORDS.some((keyword) => value.includes(keyword))) return "audio";
    if (IMAGE_KEYWORDS.some((keyword) => value.includes(keyword))) return "image";
    return "text";
}

function findChannelModel(config: AiConfig, value: string): { channel: ModelChannel; model: ChannelModel } | null {
    const decoded = decodeChannelModel(value);
    const name = decoded?.model || value;
    const channel = decoded ? config.channels.find((item) => item.id === decoded.channelId) : config.channels.find((item) => item.models.some((model) => model.name === name));
    const model = channel?.models.find((item) => item.name === name);
    return channel && model ? { channel, model } : null;
}

export function modelCapabilityOf(config: AiConfig, value: string): ModelCapability | undefined {
    return findChannelModel(config, value)?.model.capability;
}

export function modelMatchesCapability(config: AiConfig, value: string, capability?: ModelCapability) {
    if (!capability) return true;
    return modelCapabilityOf(config, value) === capability;
}

export function resolveModelForCapability(config: AiConfig, currentModel: string | undefined, capability: ModelCapability) {
    const defaultModel = capability === "image" ? config.imageModel : capability === "video" ? config.videoModel : capability === "audio" ? config.audioModel : config.textModel;
    if (currentModel && modelMatchesCapability(config, currentModel, capability)) return currentModel;
    if (defaultModel && modelMatchesCapability(config, defaultModel, capability)) return defaultModel;
    return "";
}

export function selectableModelsByCapability(config: AiConfig, capability?: ModelCapability) {
    const selectedChannel = config.channels.find((channel) => channel.id === config.platformChannelId);
    const channels = selectedChannel ? [selectedChannel] : config.channels;
    if (!capability) return selectedChannel ? modelOptionsFromChannels(channels) : config.models;
    return modelOptionsForCapability(channels, capability);
}

/** The user script (if any) attached to a model; empty string means use the system default call. */
export function resolveModelScript(config: AiConfig, value: string) {
    return findChannelModel(config, value)?.model.script?.trim() || "";
}

function isAiConfigReady(config: AiConfig, model: string) {
    const channel = resolveModelChannel(config, model);
    return Boolean(model.trim() && channel.baseUrl.trim() && channel.apiKey.trim());
}

export const useConfigStore = create<ConfigStore>()(
    persist(
        (set, get) => ({
            config: defaultConfig,
            webdav: defaultWebdavSyncConfig,
            platformModelStatus: "loading",
            platformModelError: "",
            isConfigOpen: false,
            configTab: "channels",
            shouldPromptContinue: false,
            updateConfig: (key, value) => {
                if (PLATFORM_MANAGED_CONFIG_KEYS.has(key)) return;
                set((state) => ({
                    config: {
                        ...state.config,
                        [key]: value,
                    },
                }));
            },
            setPlatformModelLoading: () => set({ platformModelStatus: "loading", platformModelError: "" }),
            setPlatformModelError: (platformModelError) =>
                set((state) => ({
                    platformModelStatus: "error",
                    platformModelError,
                    config: {
                        ...state.config,
                        channelMode: "local",
                        baseUrl: OPENAI_BASE_URL,
                        apiKey: "",
                        apiFormat: "openai",
                        channels: [],
                        platformChannelId: "",
                        models: [],
                        model: "",
                        imageModel: "",
                        videoModel: "",
                        textModel: "",
                        audioModel: "",
                    },
                })),
            applyPlatformChannels: (channels, platformModelError = "") =>
                set((state) => {
                    const models = modelOptionsFromChannels(channels);
                    const platformChannelId = channels.some((channel) => channel.id === state.config.platformChannelId) ? state.config.platformChannelId : channels[0]?.id || "";
                    const selectedChannel = channels.find((channel) => channel.id === platformChannelId);
                    const selectionChannels = selectedChannel ? [selectedChannel] : channels;
                    const imageOptions = modelOptionsForCapability(selectionChannels, "image");
                    const videoOptions = modelOptionsForCapability(selectionChannels, "video");
                    const currentImage = normalizeModelOptionValue(state.config.imageModel, channels);
                    const currentVideo = normalizeModelOptionValue(state.config.videoModel, channels);
                    const imageModel = imageOptions.includes(currentImage) ? currentImage : imageOptions[0] || "";
                    const videoModel = videoOptions.includes(currentVideo) ? currentVideo : videoOptions[0] || "";
                    const activeChannel = selectedChannel || channels[0];
                    return {
                        platformModelStatus: models.length ? "ready" : "empty",
                        platformModelError,
                        config: {
                            ...state.config,
                            channelMode: "local",
                            platformChannelId,
                            apiFormat: activeChannel?.apiFormat || "openai",
                            baseUrl: activeChannel?.baseUrl || OPENAI_BASE_URL,
                            apiKey: activeChannel?.apiKey || "",
                            channels,
                            models,
                            model: imageModel || videoModel,
                            imageModel,
                            videoModel,
                            textModel: "",
                            audioModel: "",
                        },
                    };
                }),
            selectPlatformChannel: (platformChannelId) =>
                set((state) => {
                    const channel = state.config.channels.find((item) => item.id === platformChannelId);
                    if (!channel) return {};
                    const imageModel = modelOptionsForCapability([channel], "image")[0] || "";
                    const videoModel = modelOptionsForCapability([channel], "video")[0] || "";
                    return {
                        config: {
                            ...state.config,
                            platformChannelId: channel.id,
                            apiFormat: channel.apiFormat,
                            baseUrl: channel.baseUrl || OPENAI_BASE_URL,
                            apiKey: channel.apiKey || "",
                            model: imageModel || videoModel,
                            imageModel,
                            videoModel,
                        },
                    };
                }),
            updateWebdavConfig: (key, value) =>
                set((state) => ({
                    webdav: {
                        ...state.webdav,
                        [key]: value,
                    },
                })),
            isAiConfigReady: (config, model) => isAiConfigReady(config, model),
            openConfigDialog: (shouldPromptContinue = false, configTab = "channels") => set({ isConfigOpen: true, shouldPromptContinue, configTab }),
            setConfigDialogOpen: (isConfigOpen) => set({ isConfigOpen }),
            clearPromptContinue: () => set({ shouldPromptContinue: false }),
        }),
        {
            name: CONFIG_STORE_KEY,
            partialize: (state) => ({
                config: {
                    ...state.config,
                    baseUrl: "",
                    apiKey: "",
                    channels: [],
                    model: "",
                    imageModel: "",
                    videoModel: "",
                    textModel: "",
                    audioModel: "",
                    models: [],
                },
                webdav: { ...state.webdav, password: "" },
            }),
            merge: (persisted, current) => {
                const persistedState = (persisted || {}) as Partial<ConfigStore>;
                const persistedConfig = (persistedState.config || {}) as Partial<AiConfig>;
                const persistedWebdav = (persistedState.webdav || {}) as Partial<WebdavSyncConfig>;
                const config = { ...defaultConfig, ...persistedConfig, baseUrl: defaultConfig.baseUrl, apiKey: "", channels: [] };
                return {
                    ...current,
                    webdav: { ...defaultWebdavSyncConfig, ...persistedWebdav },
                    config: {
                        ...config,
                        channelMode: "local",
                        apiFormat: normalizeApiFormat(config.apiFormat),
                        platformChannelId: typeof config.platformChannelId === "string" ? config.platformChannelId : "",
                        channels: [],
                        models: [],
                        model: "",
                        imageModel: "",
                        videoModel: "",
                        textModel: "",
                        audioModel: "",
                        audioVoice: config.audioVoice || defaultConfig.audioVoice,
                        audioFormat: config.audioFormat || defaultConfig.audioFormat,
                        audioSpeed: config.audioSpeed || defaultConfig.audioSpeed,
                        audioInstructions: config.audioInstructions || "",
                        reasoningEffort: config.reasoningEffort || "auto",
                        videoSeconds: config.videoSeconds || "6",
                        vquality: config.vquality || "720",
                        videoGenerateAudio: config.videoGenerateAudio || "true",
                        videoWatermark: config.videoWatermark || "false",
                        canvasImageCount: config.canvasImageCount || "3",
                    },
                };
            },
        },
    ),
);

export function useEffectiveConfig() {
    const config = useConfigStore((state) => state.config);
    return useMemo(() => ({ ...config, channelMode: "local" as const }), [config]);
}

/** Normalize a mixed list of raw model names or model objects into deduped ChannelModel entries. */
export function normalizeChannelModels(models: Array<string | ChannelModel> | undefined): ChannelModel[] {
    const seen = new Set<string>();
    const result: ChannelModel[] = [];
    for (const item of models || []) {
        const name = (typeof item === "string" ? item : item?.name || "").trim();
        if (!name || seen.has(name)) continue;
        seen.add(name);
        const capability = typeof item === "string" ? guessCapability(name) : item.capability || guessCapability(name);
        const script = typeof item === "string" ? undefined : item.script?.trim() || undefined;
        const video = typeof item === "string" ? undefined : item.video;
        result.push({ name, capability, script, video });
    }
    return result;
}

export function createModelChannel(channel?: Partial<ModelChannel>): ModelChannel {
    const apiFormat = normalizeApiFormat(channel?.apiFormat);
    return {
        id: channel?.id?.trim() || nanoid(),
        name: channel?.name?.trim() || i18n.t("config.channels.newName"),
        baseUrl: channel?.baseUrl?.trim() || defaultBaseUrlForApiFormat(apiFormat),
        apiKey: channel?.apiKey || "",
        apiFormat,
        models: normalizeChannelModels(channel?.models),
        videoAccountTokenId: channel?.videoAccountTokenId?.trim() || undefined,
    };
}

export function encodeChannelModel(channelId: string, model: string) {
    return `${channelId}${CHANNEL_MODEL_SEPARATOR}${model.trim()}`;
}

export function isChannelModelValue(value: string) {
    return value.includes(CHANNEL_MODEL_SEPARATOR);
}

export function decodeChannelModel(value: string) {
    const index = value.indexOf(CHANNEL_MODEL_SEPARATOR);
    if (index < 0) return null;
    return { channelId: value.slice(0, index), model: value.slice(index + CHANNEL_MODEL_SEPARATOR.length) };
}

export function modelOptionName(value: string) {
    return decodeChannelModel(value)?.model || value;
}

export function modelOptionLabel(config: AiConfig, value: string) {
    const decoded = decodeChannelModel(value);
    if (!decoded) return value;
    const channel = config.channels.find((item) => item.id === decoded.channelId);
    return channel ? `${decoded.model}（${channel.name}）` : decoded.model;
}

export function modelOptionsFromChannels(channels: ModelChannel[]) {
    return uniqueModelOptions(channels.flatMap((channel) => channel.models.map((model) => encodeChannelModel(channel.id, model.name))));
}

function modelOptionsForCapability(channels: ModelChannel[], capability: ModelCapability) {
    return uniqueModelOptions(channels.flatMap((channel) => channel.models.filter((model) => model.capability === capability).map((model) => encodeChannelModel(channel.id, model.name))));
}

export function normalizeModelOptionValue(value: string | undefined, channels: ModelChannel[]) {
    const model = (value || "").trim();
    if (!model) return "";
    const decoded = decodeChannelModel(model);
    if (decoded) {
        const channel = channels.find((item) => item.id === decoded.channelId);
        return channel && channel.models.some((item) => item.name === decoded.model) ? model : "";
    }
    const channel = channels.find((item) => item.models.some((entry) => entry.name === model)) || channels[0];
    return channel && channel.models.some((item) => item.name === model) ? encodeChannelModel(channel.id, model) : model;
}

export function resolveModelChannel(config: AiConfig, value: string) {
    const decoded = decodeChannelModel(value);
    const model = decoded?.model || value;
    const matched = decoded ? config.channels.find((channel) => channel.id === decoded.channelId) : config.channels.find((channel) => channel.models.some((item) => item.name === model));
    return (
        matched ||
        config.channels[0] ||
        createModelChannel({
            id: "default",
            name: i18n.t("config.channels.defaultName"),
            baseUrl: config.baseUrl,
            apiKey: config.apiKey,
            apiFormat: config.apiFormat,
            models: config.models.map(modelOptionName).map((name) => ({ name, capability: guessCapability(name) })),
        })
    );
}

export function resolveModelRequestConfig(config: AiConfig, value: string) {
    const channel = resolveModelChannel(config, value);
    return {
        ...config,
        model: modelOptionName(value || config.model),
        baseUrl: channel.baseUrl,
        apiKey: channel.apiKey,
        apiFormat: channel.apiFormat,
        videoAccountTokenId: channel.videoAccountTokenId,
    };
}

export function defaultBaseUrlForApiFormat(apiFormat: ApiCallFormat) {
    if (apiFormat === "gemini") return GEMINI_BASE_URL;
    return OPENAI_BASE_URL;
}

function normalizeApiFormat(apiFormat: unknown): ApiCallFormat {
    return apiFormat === "gemini" ? apiFormat : "openai";
}

function uniqueModelOptions(models: string[]) {
    return Array.from(new Set((models || []).map((model) => model.trim()).filter(Boolean)));
}

export function buildApiUrl(baseUrl: string, path: string) {
    const normalizedBaseUrl = baseUrl.trim().replace(/\/+$/, "");
    const lowerBaseUrl = normalizedBaseUrl.toLowerCase();
    const apiBaseUrl = lowerBaseUrl.endsWith("/v1") ? normalizedBaseUrl : `${normalizedBaseUrl}/v1`;
    return `${apiBaseUrl}${path}`;
}
