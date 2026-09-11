// @vitest-environment jsdom

import { afterEach, describe, expect, it, vi } from "vitest";

import { loadPlatformModelChannels, loadVideoAccountChannels } from "../platform-models";

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

    it("loads dedicated video accounts with opaque selectors and catalog constraints", async () => {
        const fetchMock = vi.fn().mockResolvedValue(
            new Response(
                JSON.stringify({
                    data: [
                        {
                            id: "minimax-h3-01",
                            display_name: "MiniMax H3",
                            private_group_key: "vca_account",
                            group: "minimax-h3",
                            available: true,
                            durations_seconds: [6, 10],
                            ratios: ["16:9"],
                            max_images: -1,
                            pricing: { mode: "per_second" },
                        },
                    ],
                    private_groups: [{ key: "vca_account", name: "CTMOAI 主账号" }],
                }),
                { status: 200, headers: { "Content-Type": "application/json" } },
            ),
        );
        vi.stubGlobal("fetch", fetchMock);

        const channels = await loadVideoAccountChannels("dashboard-token", new AbortController().signal);

        expect(fetchMock).toHaveBeenCalledWith(
            "/api/video-creation/catalog",
            expect.objectContaining({
                credentials: "include",
                headers: expect.objectContaining({ Authorization: "Bearer dashboard-token" }),
            }),
        );
        expect(channels).toHaveLength(1);
        expect(channels[0]).toMatchObject({
            id: "video-account-vca_account",
            name: "CTMOAI 主账号",
            videoAccountTokenId: "vca_account",
        });
        expect(channels[0].models[0]).toMatchObject({
            name: "minimax-h3-01",
            capability: "video",
            video: { durationsSeconds: [6, 10], maxImages: -1, pricingMode: "per_second" },
        });
    });
});
