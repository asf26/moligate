import { resolveModelVideoMetadata, type AiConfig, type VideoModelMetadata } from "@/stores/use-config-store";

/**
 * The video controls the workspace can offer for one model, derived entirely
 * from the capability metadata the gateway publishes for that model. A field
 * that is empty or zero means "not supported", which is what lets the UI hide
 * a control instead of sending a request the gateway would reject.
 */
export type VideoModelCapabilities = {
    /** False for models served by a plain relay channel, which have no catalog data. */
    known: boolean;
    metadata?: VideoModelMetadata;
    /** Upstream integration serving the model: "minimax-h3" or "seedance". */
    family: string;
    /** Fixed output resolution; CTMOAI bakes it into the model id. */
    resolution: string;
    ratios: string[];
    /** Size value to submit per ratio. Empty when the model takes no size. */
    ratioSizes: Record<string, string>;
    durationsSeconds: number[];
    /**
     * Reference limits as published by the gateway. `0` means the model rejects
     * that asset kind, a positive number is the exact bound, and `-1` (the
     * gateway's marker for a bound the upstream catalog did not publish) is not
     * a bound at all. The three cases are kept apart rather than collapsed.
     */
    maxImages?: number;
    maxVideos?: number;
    maxAudios?: number;
    audioRequiresImage: boolean;
    requiresImage: boolean;
    supportsFirstLastFrame: boolean;
    /** "per_second" or "per_task"; the amount below is priced accordingly. */
    pricingMode: string;
    pricingAmount: number;
    pricingCurrency: string;
};

/**
 * Reference-image limit for models without catalog data. Plain channels have no
 * advertised limit, so keep the value the workspace used before capabilities
 * were available.
 */
export const fallbackReferenceImageLimit = 7;

/**
 * Caps used when the catalog publishes no bound (`-1`). They match what the
 * gateway accepted before it started reporting limits, and keep the request
 * well inside the relay's own reference ceiling.
 */
const unboundedReferenceLimits = { image: 9, video: 3, audio: 3 } as const;

/**
 * Per-asset upload ceiling in MB. Mirrors the gateway's own limits
 * (`controller/video_account_media.go`), which in turn match what CTMOAI's
 * console enforces, so a file that passes here is not rejected later.
 */
export const referenceUploadLimits = { image: 10, video: 50, audio: 30 } as const;

export function resolveVideoModelCapabilities(config: AiConfig, model: string): VideoModelCapabilities {
    const metadata = resolveModelVideoMetadata(config, model);
    const ratios = uniqueStrings(metadata?.ratios);
    return {
        known: Boolean(metadata),
        metadata,
        family: (metadata?.family || "").trim(),
        resolution: (metadata?.resolution || "").trim(),
        ratios,
        ratioSizes: normalizeRatioSizes(metadata?.ratioSizes, ratios),
        durationsSeconds: normalizeDurations(metadata?.durationsSeconds),
        maxImages: finiteLimit(metadata?.maxImages),
        maxVideos: finiteLimit(metadata?.maxVideos),
        maxAudios: finiteLimit(metadata?.maxAudios),
        audioRequiresImage: Boolean(metadata?.audioRequiresImage),
        requiresImage: Boolean(metadata?.requiresImage),
        supportsFirstLastFrame: Boolean(metadata?.supportsFirstLastFrame),
        pricingMode: (metadata?.pricingMode || "").trim(),
        pricingAmount: typeof metadata?.pricingAmount === "number" && Number.isFinite(metadata.pricingAmount) && metadata.pricingAmount > 0 ? metadata.pricingAmount : 0,
        pricingCurrency: (metadata?.pricingCurrency || "").trim(),
    };
}

const currencySymbols: Record<string, string> = { CNY: "¥", RMB: "¥", USD: "$", EUR: "€" };

function formatMoney(amount: number, currency: string) {
    const rounded = Math.round(amount * 100) / 100;
    // Whole amounts stay bare; anything fractional keeps two decimals so a price
    // reads like a price tag.
    const text = Number.isInteger(rounded) ? String(rounded) : rounded.toFixed(2);
    const code = currency.toUpperCase();
    const symbol = currencySymbols[code];
    return symbol ? `${symbol}${text}` : `${code ? `${code} ` : ""}${text}`;
}

/**
 * Price of the generation the current settings would submit. Per-second models
 * are quoted per second plus the total for the selected duration; per-task
 * models are quoted flat, because the duration does not change the charge.
 * Returns null when the model publishes no price.
 */
export function videoModelPrice(capabilities: VideoModelCapabilities, seconds: string) {
    if (!capabilities.pricingAmount) return null;
    const currency = capabilities.pricingCurrency || "CNY";
    if (capabilities.pricingMode.toLowerCase() !== "per_second") {
        return { unit: formatMoney(capabilities.pricingAmount, currency), total: "" };
    }
    const duration = Math.max(1, Math.floor(Number(seconds) || 0));
    return {
        unit: formatMoney(capabilities.pricingAmount, currency),
        total: formatMoney(capabilities.pricingAmount * duration, currency),
    };
}

/**
 * How many reference images may be attached. Images are the primary reference
 * kind, so a catalog that publishes no bound keeps the historical limit instead
 * of blocking the input outright.
 */
export function referenceImageLimit(capabilities: VideoModelCapabilities) {
    if (!capabilities.known) return fallbackReferenceImageLimit;
    return boundedLimit(capabilities.maxImages, unboundedReferenceLimits.image);
}

/**
 * How many reference videos or audios may be attached. Only a model that is
 * known to accept them gets the input.
 */
export function referenceMediaLimit(capabilities: VideoModelCapabilities, kind: "video" | "audio") {
    if (!capabilities.known) return 0;
    return boundedLimit(kind === "video" ? capabilities.maxVideos : capabilities.maxAudios, unboundedReferenceLimits[kind]);
}

/** Picks the ratio to submit, preferring the requested one when the model allows it. */
export function normalizeVideoModelRatio(capabilities: VideoModelCapabilities, requested: string) {
    const value = (requested || "").trim();
    if (!capabilities.known) return value;
    if (!capabilities.ratios.length) return "";
    return capabilities.ratios.includes(value) ? value : capabilities.ratios[0];
}

/**
 * Picks the duration to submit. Models advertise an explicit list, and some only
 * accept a handful of values, so snap to the closest supported one rather than
 * letting the gateway reject the request.
 */
export function normalizeVideoModelSeconds(capabilities: VideoModelCapabilities, requested: string) {
    const values = capabilities.durationsSeconds;
    const seconds = Math.floor(Number(requested) || 0);
    if (!capabilities.known || !values.length) return String(seconds > 0 ? seconds : Number(requested) || 0);
    if (values.includes(seconds)) return String(seconds);
    return String(values.reduce((closest, item) => (Math.abs(item - seconds) < Math.abs(closest - seconds) ? item : closest), values[0]));
}

/** Size value the model expects for a ratio; empty when it takes none. */
export function videoModelSizeForRatio(capabilities: VideoModelCapabilities, ratio: string) {
    return capabilities.ratioSizes[(ratio || "").trim()] || "";
}

/** Turns a "16:9" style ratio into the width/height pair used by the size previews. */
export function ratioDimensions(ratio: string) {
    const match = (ratio || "").trim().match(/^(\d+)\s*:\s*(\d+)$/);
    if (!match) return { width: 0, height: 0 };
    const width = Number(match[1]);
    const height = Number(match[2]);
    if (!width || !height) return { width: 0, height: 0 };
    return { width, height };
}

function finiteLimit(value: number | undefined) {
    return typeof value === "number" && Number.isFinite(value) ? Math.floor(value) : undefined;
}

/** Resolves a published limit, treating a negative value as "no published bound". */
function boundedLimit(value: number | undefined, unbounded: number) {
    if (value === undefined || value < 0) return unbounded;
    return value;
}

function uniqueStrings(values: string[] | undefined) {
    const result: string[] = [];
    for (const value of values || []) {
        const item = String(value || "").trim();
        if (item && !result.includes(item)) result.push(item);
    }
    return result;
}

function normalizeDurations(values: number[] | undefined) {
    const result = (values || []).filter((value) => Number.isInteger(value) && value > 0 && value <= 3600);
    return Array.from(new Set(result)).sort((left, right) => left - right);
}

function normalizeRatioSizes(values: Record<string, string> | undefined, ratios: string[]) {
    const result: Record<string, string> = {};
    if (!values) return result;
    for (const ratio of ratios) {
        const size = String(values[ratio] || "").trim();
        if (size) result[ratio] = size;
    }
    return result;
}
