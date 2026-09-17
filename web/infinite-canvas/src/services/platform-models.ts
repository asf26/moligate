import { createModelChannel, guessCapability, type ChannelModel, type ModelChannel, type VideoModelMetadata } from "@/stores/use-config-store";

/** A model entry as returned by the gateway's canvas config. */
export type PlatformCanvasModel = {
    name?: string;
    capability?: ChannelModel["capability"];
    /** Present only when the model is served by a dedicated video account. */
    video?: PlatformVideoMetadata;
};

/**
 * Wire shape of the gateway's per-model video capability metadata. The account
 * selector travels in the same object as the capability fields, so the
 * workspace can never apply the capabilities of one account to another.
 */
export type PlatformVideoMetadata = {
    video_account_token_id?: string;
    group?: string;
    family?: string;
    resolution?: string;
    durations_seconds?: number[];
    ratios?: string[];
    sizes?: string[];
    ratio_sizes?: Record<string, string>;
    max_images?: number;
    max_videos?: number;
    max_audios?: number;
    audio_requires_image?: boolean;
    requires_image?: boolean;
    supports_first_last_frame?: boolean;
    pricing_mode?: string;
    pricing_amount?: number;
    pricing_currency?: string;
};

export type PlatformCanvasGroup = {
    id: string;
    name: string;
    api_key: string;
    key_id?: number;
    key_name?: string;
    group_id?: string;
    group_name?: string;
    models?: PlatformCanvasModel[];
};

type GatewayModel = {
    id?: string;
    supported_endpoint_types?: string[];
};

type GatewayModelPayload = {
    success?: boolean;
    message?: string;
    data?: GatewayModel[];
    error?: { message?: string };
};

export type PlatformChannelResult = {
    channels: ModelChannel[];
    warning: string;
};

const platformApiBaseUrl = `${window.location.origin}/v1`;
const imageEndpoint = "image-generation";
const videoEndpoint = "openai-video";

export async function loadPlatformModelChannels(groups: PlatformCanvasGroup[], signal: AbortSignal): Promise<PlatformChannelResult> {
    if (!groups.length) return { channels: [], warning: "" };

    const results = await Promise.allSettled(groups.map((group) => loadPlatformGroup(group, signal)));
    const channels: ModelChannel[] = [];
    const failures: string[] = [];
    results.forEach((result, index) => {
        if (result.status === "fulfilled") {
            channels.push(result.value);
            return;
        }
        const reason = result.reason instanceof Error ? result.reason.message : String(result.reason || "");
        failures.push(`${groups[index].name || groups[index].id}: ${reason}`);
    });

    if (!channels.length && failures.length) throw new Error(failures.join("; "));
    return { channels, warning: failures.join("; ") };
}

async function loadPlatformGroup(group: PlatformCanvasGroup, signal: AbortSignal) {
    const models: ChannelModel[] = [];
    const seen = new Set<string>();
    const addModel = (entry: PlatformCanvasModel) => {
        const name = entry.name?.trim() || "";
        if (!name || seen.has(name)) return;
        const capability = entry.capability || guessCapability(name);
        if (capability !== "image" && capability !== "video") return;
        seen.add(name);
        const source = entry.video;
        const videoAccountTokenId = source?.video_account_token_id?.trim() || "";
        // A plain gateway model has no dedicated account: it is relayed with the
        // key alone and needs no video capability metadata.
        if (!source || !videoAccountTokenId) {
            models.push({ name, capability });
            return;
        }
        const video: VideoModelMetadata = {
            group: source.group,
            family: source.family,
            resolution: source.resolution,
            durationsSeconds: source.durations_seconds,
            ratios: source.ratios,
            sizes: source.sizes,
            ratioSizes: source.ratio_sizes,
            maxImages: source.max_images,
            maxVideos: source.max_videos,
            maxAudios: source.max_audios,
            audioRequiresImage: source.audio_requires_image,
            requiresImage: source.requires_image,
            supportsFirstLastFrame: source.supports_first_last_frame,
            pricingMode: source.pricing_mode,
            pricingAmount: source.pricing_amount,
            pricingCurrency: source.pricing_currency,
        };
        models.push({ name, capability, video, videoAccountTokenId });
    };

    if (Array.isArray(group.models)) {
        for (const item of group.models) addModel(item);
    } else {
        const response = await fetch(`${platformApiBaseUrl}/models`, {
            headers: { Accept: "application/json", Authorization: `Bearer ${group.api_key}` },
            signal,
        });
        let payload: GatewayModelPayload | undefined;
        try {
            payload = (await response.json()) as GatewayModelPayload;
        } catch {
            payload = undefined;
        }
        if (!response.ok || payload?.success === false) {
            throw new Error(payload?.message || payload?.error?.message || `HTTP ${response.status}`);
        }
        for (const item of payload?.data || []) {
            const name = item.id?.trim() || "";
            if (!name || seen.has(name)) continue;
            const endpoints = new Set((item.supported_endpoint_types || []).map((endpoint) => endpoint.toLowerCase()));
            let capability: ChannelModel["capability"] | undefined;
            if (endpoints.has(imageEndpoint)) capability = "image";
            if (endpoints.has(videoEndpoint)) capability = "video";
            addModel({ name, capability });
        }
    }

    // The gateway already composes a human-readable key/group label. Do not
    // append the internal token id: it is only a stable routing identifier.
    const displayName = group.name.trim() || group.key_name?.trim() || group.group_name?.trim() || group.id;
    return createModelChannel({
        id: `platform-${encodeURIComponent(group.id)}`,
        name: displayName,
        baseUrl: platformApiBaseUrl,
        apiKey: group.api_key,
        apiFormat: "openai",
        models,
    });
}
