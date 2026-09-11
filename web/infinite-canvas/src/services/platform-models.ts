import { createModelChannel, guessCapability, type ChannelModel, type ModelChannel } from "@/stores/use-config-store";

export type PlatformCanvasGroup = {
    id: string;
    name: string;
    api_key: string;
    key_id?: number;
    key_name?: string;
    group_id?: string;
    group_name?: string;
    models?: Array<{ name?: string; capability?: ChannelModel["capability"] }>;
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
    const addModel = (nameValue: string | undefined, capabilityValue?: ChannelModel["capability"]) => {
        const name = nameValue?.trim() || "";
        if (!name || seen.has(name)) return;
        const capability = capabilityValue || guessCapability(name);
        if (capability !== "image" && capability !== "video") return;
        seen.add(name);
        models.push({ name, capability });
    };

    if (Array.isArray(group.models)) {
        for (const item of group.models) addModel(item.name, item.capability);
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
            addModel(name, capability);
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
