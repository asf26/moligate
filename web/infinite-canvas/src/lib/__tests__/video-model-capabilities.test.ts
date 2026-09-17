// @vitest-environment jsdom

import { describe, expect, it } from "vitest";

import { fallbackReferenceImageLimit, normalizeVideoModelRatio, normalizeVideoModelSeconds, ratioDimensions, referenceImageLimit, referenceMediaLimit, resolveVideoModelCapabilities, videoModelSizeForRatio } from "../video-model-capabilities";
import type { AiConfig } from "@/stores/use-config-store";

const h3Ratios = ["16:9", "9:16", "1:1", "2:3", "3:2", "3:4", "4:3", "21:9"];

function configWithVideo(video: Record<string, unknown> | undefined): AiConfig {
    return {
        baseUrl: "https://gateway.example/v1",
        apiKey: "sk-canvas-key",
        apiFormat: "openai",
        model: "vca::model-under-test",
        videoModel: "",
        videoSeconds: "6",
        size: "16:9",
        channels: [{ id: "vca", name: "key · group", baseUrl: "https://gateway.example/v1", apiKey: "sk-canvas-key", apiFormat: "openai", models: [{ name: "model-under-test", capability: "video", videoAccountTokenId: "vca_account", video }] }],
    } as unknown as AiConfig;
}

describe("resolveVideoModelCapabilities", () => {
    it("reads every control the panel needs from the model metadata", () => {
        const capabilities = resolveVideoModelCapabilities(
            configWithVideo({
                resolution: "768p",
                durationsSeconds: [4, 5, 6],
                ratios: h3Ratios,
                ratioSizes: { "16:9": "1376x768", "9:16": "768x1376" },
                maxImages: 9,
                maxVideos: 3,
                maxAudios: 3,
                supportsFirstLastFrame: true,
                requiresImage: true,
            }),
            "vca::model-under-test",
        );

        expect(capabilities.known).toBe(true);
        expect(capabilities.resolution).toBe("768p");
        expect(capabilities.durationsSeconds).toEqual([4, 5, 6]);
        expect(capabilities.ratios).toEqual(h3Ratios);
        expect(capabilities.requiresImage).toBe(true);
        expect(capabilities.supportsFirstLastFrame).toBe(true);
        // ratio_sizes drives the submitted size, so it must stay aligned.
        expect(videoModelSizeForRatio(capabilities, "9:16")).toBe("768x1376");
        expect(videoModelSizeForRatio(capabilities, "1:1")).toBe("");
    });

    it("reports an unknown model so the panel keeps the generic controls", () => {
        const capabilities = resolveVideoModelCapabilities(configWithVideo(undefined), "vca::model-under-test");
        expect(capabilities.known).toBe(false);
        expect(referenceImageLimit(capabilities)).toBe(fallbackReferenceImageLimit);
        // Without a catalog there is no evidence the model takes extra assets.
        expect(referenceMediaLimit(capabilities, "video")).toBe(0);
        expect(referenceMediaLimit(capabilities, "audio")).toBe(0);
    });

    it("tells an explicit zero limit apart from an unpublished one", () => {
        const capabilities = resolveVideoModelCapabilities(configWithVideo({ resolution: "768p", maxImages: 4, maxVideos: 0, maxAudios: 0 }), "vca::model-under-test");
        // 0 is a real bound: the model rejects that asset kind.
        expect(referenceImageLimit(capabilities)).toBe(4);
        expect(referenceMediaLimit(capabilities, "video")).toBe(0);
        expect(referenceMediaLimit(capabilities, "audio")).toBe(0);

        // -1 is the gateway's marker for "upstream published no bound", so the
        // inputs stay available with the historical caps.
        const unbounded = resolveVideoModelCapabilities(configWithVideo({ resolution: "768p", maxImages: -1, maxVideos: -1, maxAudios: -1 }), "vca::model-under-test");
        expect(referenceImageLimit(unbounded)).toBe(9);
        expect(referenceMediaLimit(unbounded, "video")).toBe(3);
        expect(referenceMediaLimit(unbounded, "audio")).toBe(3);

        const missing = resolveVideoModelCapabilities(configWithVideo({ resolution: "768p" }), "vca::model-under-test");
        expect(referenceImageLimit(missing)).toBe(9);
        expect(referenceMediaLimit(missing, "video")).toBe(3);
    });
});

describe("normalizers", () => {
    it("snaps the ratio onto a supported one", () => {
        const capabilities = resolveVideoModelCapabilities(configWithVideo({ ratios: ["16:9", "9:16"] }), "vca::model-under-test");
        expect(normalizeVideoModelRatio(capabilities, "9:16")).toBe("9:16");
        // A leftover pixel size from a generic model must not reach the gateway.
        expect(normalizeVideoModelRatio(capabilities, "1280x720")).toBe("16:9");
    });

    it("snaps the duration onto the closest supported value", () => {
        const capabilities = resolveVideoModelCapabilities(configWithVideo({ durationsSeconds: [10, 15] }), "vca::model-under-test");
        expect(normalizeVideoModelSeconds(capabilities, "10")).toBe("10");
        expect(normalizeVideoModelSeconds(capabilities, "6")).toBe("10");
        expect(normalizeVideoModelSeconds(capabilities, "14")).toBe("15");
    });

    it("leaves the requested values alone when nothing is known", () => {
        const capabilities = resolveVideoModelCapabilities(configWithVideo(undefined), "vca::model-under-test");
        expect(normalizeVideoModelRatio(capabilities, "1280x720")).toBe("1280x720");
        expect(normalizeVideoModelSeconds(capabilities, "6")).toBe("6");
    });

    it("parses ratio strings into preview dimensions", () => {
        expect(ratioDimensions("16:9")).toEqual({ width: 16, height: 9 });
        expect(ratioDimensions("21:9")).toEqual({ width: 21, height: 9 });
        expect(ratioDimensions("1280x720")).toEqual({ width: 0, height: 0 });
    });
});
