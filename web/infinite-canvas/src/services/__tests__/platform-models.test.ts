// @vitest-environment jsdom

import { afterEach, describe, expect, it, vi } from "vitest";

import { loadPlatformModelChannels } from "../platform-models";

describe("loadPlatformModelChannels", () => {
    afterEach(() => vi.unstubAllGlobals());

    it("loads each group through its own key and keeps only creation models", async () => {
        const fetchMock = vi
            .fn()
            .mockResolvedValueOnce(
                new Response(
                    JSON.stringify({
                        success: true,
                        data: [
                            { id: "gpt-image-2", supported_endpoint_types: ["image-generation", "openai"] },
                            { id: "seedance2.0-stable-full-720p", supported_endpoint_types: ["openai-video"] },
                            { id: "gpt-5.4", supported_endpoint_types: ["openai"] },
                        ],
                    }),
                    { status: 200, headers: { "Content-Type": "application/json" } },
                ),
            )
            .mockResolvedValueOnce(new Response(JSON.stringify({ success: false, message: "group unavailable" }), { status: 503, headers: { "Content-Type": "application/json" } }));
        vi.stubGlobal("fetch", fetchMock);

        const result = await loadPlatformModelChannels(
            [
                { id: "default", name: "默认分组", api_key: "sk-default" },
                { id: "vip", name: "VIP", api_key: "sk-vip" },
            ],
            new AbortController().signal,
        );

        expect(fetchMock).toHaveBeenNthCalledWith(1, `${window.location.origin}/v1/models`, expect.objectContaining({ headers: expect.objectContaining({ Authorization: "Bearer sk-default" }) }));
        expect(fetchMock).toHaveBeenNthCalledWith(2, `${window.location.origin}/v1/models`, expect.objectContaining({ headers: expect.objectContaining({ Authorization: "Bearer sk-vip" }) }));
        expect(result.channels).toHaveLength(1);
        expect(result.channels[0].name).toBe("默认分组");
        expect(result.channels[0].models).toEqual([
            { name: "gpt-image-2", capability: "image" },
            { name: "seedance2.0-stable-full-720p", capability: "video" },
        ]);
        expect(result.warning).toContain("VIP: group unavailable");
    });

    it("treats an account with no creation groups as a valid empty state", async () => {
        const fetchMock = vi.fn();
        vi.stubGlobal("fetch", fetchMock);
        await expect(loadPlatformModelChannels([], new AbortController().signal)).resolves.toEqual({ channels: [], warning: "" });
        expect(fetchMock).not.toHaveBeenCalled();
    });

    it("uses platform-provided model capabilities without exposing another endpoint request", async () => {
        const fetchMock = vi.fn();
        vi.stubGlobal("fetch", fetchMock);

        const result = await loadPlatformModelChannels(
            [
                {
                    id: "token-7",
                    name: "创作 Key · 默认分组",
                    api_key: "sk-managed",
                    models: [
                        { name: "gpt-image-2", capability: "image" },
                        { name: "seedance2.0-stable-full-720p", capability: "video" },
                    ],
                },
            ],
            new AbortController().signal,
        );

        expect(fetchMock).not.toHaveBeenCalled();
        expect(result.channels[0].models).toEqual([
            { name: "gpt-image-2", capability: "image" },
            { name: "seedance2.0-stable-full-720p", capability: "video" },
        ]);
    });

    it("surfaces a dedicated video account's models from the key's own canvas group", async () => {
        const fetchMock = vi.fn();
        vi.stubGlobal("fetch", fetchMock);

        const result = await loadPlatformModelChannels(
            [
                {
                    id: "token-95",
                    name: "视频h3 · 视频-h3",
                    api_key: "sk-video-h3",
                    group_id: "视频-h3",
                    models: [
                        { name: "gpt-image-2", capability: "image" },
                        {
                            name: "minimax-h3-01",
                            capability: "video",
                            video: {
                                video_account_token_id: "vca_account",
                                group: "minimax-h3",
                                resolution: "768p",
                                durations_seconds: [6, 10],
                                ratios: ["16:9"],
                                ratio_sizes: { "16:9": "1376x768" },
                                max_images: 9,
                                max_videos: 3,
                                max_audios: 3,
                                audio_requires_image: false,
                                requires_image: true,
                                supports_first_last_frame: true,
                                pricing_mode: "per_second",
                            },
                        },
                        // Capability metadata without a selector is not a
                        // dedicated account and must stay a plain model.
                        { name: "seedance2.0-stable-full-720p", capability: "video", video: { durations_seconds: [5] } },
                    ],
                },
            ],
            new AbortController().signal,
        );

        // The gateway already sent the key's models, so no extra /v1/models call.
        expect(fetchMock).not.toHaveBeenCalled();
        expect(result.channels[0].apiKey).toBe("sk-video-h3");
        expect(result.channels[0].models[0]).toEqual({ name: "gpt-image-2", capability: "image" });
        expect(result.channels[0].models[1]).toMatchObject({
            name: "minimax-h3-01",
            capability: "video",
            videoAccountTokenId: "vca_account",
            video: {
                group: "minimax-h3",
                resolution: "768p",
                durationsSeconds: [6, 10],
                ratios: ["16:9"],
                ratioSizes: { "16:9": "1376x768" },
                maxImages: 9,
                maxVideos: 3,
                maxAudios: 3,
                audioRequiresImage: false,
                requiresImage: true,
                supportsFirstLastFrame: true,
                pricingMode: "per_second",
            },
        });
        expect(result.channels[0].models[2]).toEqual({ name: "seedance2.0-stable-full-720p", capability: "video" });
        expect(result.channels[0].models[2].videoAccountTokenId).toBeUndefined();
        expect(result.channels[0].models[2].video).toBeUndefined();
    });
});
