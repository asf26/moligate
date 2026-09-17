// @vitest-environment jsdom

import axios from "axios";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { requestVideoGeneration } from "../video";
import type { AiConfig } from "@/stores/use-config-store";

vi.mock("axios", () => ({
    default: {
        post: vi.fn(),
        get: vi.fn(),
        isAxiosError: vi.fn(() => false),
        isCancel: vi.fn(() => false),
    },
}));

vi.mock("@/i18n", () => ({ default: { t: (key: string) => key } }));

vi.mock("@/lib/image-utils", () => ({
    dataUrlToFile: vi.fn(async () => new File(["reference"], "reference.png", { type: "image/png" })),
}));

vi.mock("@/services/file-storage", () => ({ uploadMediaFile: vi.fn() }));

vi.mock("@/services/image-storage", () => ({
    imageToDataUrl: vi.fn(async (image: { dataUrl: string }) => image.dataUrl),
}));

vi.mock("@/stores/use-config-store", () => {
    const channelFor = (config: AiConfig, value: string) => {
        const [channelId, modelName] = value.includes("::") ? value.split("::") : ["", value];
        return config.channels?.find((channel) => channel.id === channelId || channel.models.some((model) => model.name === modelName));
    };
    const modelNameOf = (value: string) => value.split("::").at(-1) || value;
    return {
        boolConfig: vi.fn(),
        buildApiUrl: (baseUrl: string, path: string) => `${baseUrl}${path}`,
        modelOptionName: modelNameOf,
        resolveModelChannel: (config: AiConfig, value: string) => channelFor(config, value) || { id: "default", name: "default", baseUrl: config.baseUrl, apiKey: config.apiKey, apiFormat: config.apiFormat, models: [] },
        // Mirrors the real resolver: the account selector belongs to the model.
        resolveModelRequestConfig: (config: AiConfig, value: string) => {
            const channel = channelFor(config, value);
            const model = modelNameOf(value);
            return {
                ...config,
                model,
                baseUrl: channel?.baseUrl || config.baseUrl,
                apiKey: channel?.apiKey || config.apiKey,
                apiFormat: channel?.apiFormat || config.apiFormat,
                videoAccountTokenId: channel?.models.find((item) => item.name === model)?.videoAccountTokenId,
            };
        },
        resolveModelScript: () => "",
    };
});

vi.mock("../model-plugin", () => ({ runModelPlugin: vi.fn() }));

describe("requestVideoGeneration", () => {
    beforeEach(() => {
        vi.clearAllMocks();
        vi.mocked(axios.post).mockResolvedValue({ data: { id: "video_123" } });
        vi.mocked(axios.get)
            .mockResolvedValueOnce({ data: { id: "video_123", status: "completed" } })
            .mockResolvedValueOnce({ data: new Blob(["video"], { type: "video/mp4" }) });
    });

    it("sends MiniMax H3 requests with the documented image field and normalized options", async () => {
        const config = {
            baseUrl: "https://gateway.example/v1",
            apiKey: "canvas-key",
            apiFormat: "openai",
            model: "canvas::minimax-h3-original-768p",
            videoModel: "",
            videoSeconds: "2",
            size: "1:1",
        } as AiConfig;

        await requestVideoGeneration(config, "animate this image", [{ id: "reference", name: "reference.png", type: "image/png", dataUrl: "data:image/png;base64,cmVmZXJlbmNl" }]);

        const postCall = vi.mocked(axios.post).mock.calls[0];
        const body = postCall[1] as FormData;
        expect(postCall[0]).toBe("https://gateway.example/v1/videos");
        expect(body.get("model")).toBe("minimax-h3-original-768p");
        expect(body.get("seconds")).toBe("4");
        expect(body.get("aspect_ratio")).toBe("1:1");
        expect(body.get("images")).toBeInstanceOf(File);
        expect(body.get("input_reference[]")).toBeNull();
    });

    it("uploads Stable references through the platform and creates a JSON video task", async () => {
        vi.mocked(axios.post)
            .mockResolvedValueOnce({ data: { url: "https://media.example/reference.png" } })
            .mockResolvedValueOnce({ data: { id: "video_123" } });
        const config = {
            baseUrl: "https://gateway.example/v1",
            apiKey: "canvas-key",
            apiFormat: "openai",
            model: "platform-default::seedance2.0-stable-facecheck-720p",
            videoModel: "",
            videoSeconds: "3",
            size: "16:9",
        } as AiConfig;

        await requestVideoGeneration(config, "animate this portrait", [{ id: "reference", name: "reference.png", type: "image/png", dataUrl: "data:image/png;base64,cmVmZXJlbmNl" }]);

        const uploadCall = vi.mocked(axios.post).mock.calls[0];
        expect(uploadCall[0]).toBe("https://gateway.example/v1/videos/media");
        expect((uploadCall[1] as FormData).get("model")).toBe("seedance2.0-stable-facecheck-720p");
        expect((uploadCall[1] as FormData).get("type")).toBe("images");
        expect((uploadCall[1] as FormData).get("file")).toBeInstanceOf(File);

        const createCall = vi.mocked(axios.post).mock.calls[1];
        expect(createCall[0]).toBe("https://gateway.example/v1/videos");
        expect(createCall[1]).toEqual({
            model: "seedance2.0-stable-facecheck-720p",
            prompt: "animate this portrait",
            seconds: 5,
            aspect_ratio: "9:16",
            images: ["https://media.example/reference.png"],
        });
        expect(createCall[2]).toEqual(expect.objectContaining({ timeout: 90_000 }));
    });

    it("routes a dedicated account model through the key itself and its per-model selector", async () => {
        vi.mocked(axios.post)
            .mockResolvedValueOnce({ data: { url: "https://media.example/reference.png" } })
            .mockResolvedValueOnce({ data: { id: "video_123" } });
        const config = {
            baseUrl: "https://gateway.example/v1",
            apiKey: "sk-canvas-key",
            apiFormat: "openai",
            model: "vca::minimax-h3-01",
            videoModel: "",
            videoSeconds: "7",
            size: "16:9",
            channels: [
                {
                    id: "vca",
                    name: "视频h3 key · 视频-h3",
                    baseUrl: "https://gateway.example/v1",
                    // The browser only ever holds the caller's own key; the
                    // upstream CTMOAI credential stays on the server.
                    apiKey: "sk-canvas-key",
                    apiFormat: "openai",
                    models: [
                        {
                            name: "minimax-h3-01",
                            capability: "video",
                            videoAccountTokenId: "vca_account",
                            video: { durationsSeconds: [6, 10], ratios: ["16:9"] },
                        },
                    ],
                },
            ],
        } as AiConfig;

        await requestVideoGeneration(config, "animate this image", [{ id: "reference", name: "reference.png", type: "image/png", dataUrl: "data:image/png;base64,cmVmZXJlbmNl" }]);

        const uploadCall = vi.mocked(axios.post).mock.calls[0];
        expect(uploadCall[0]).toBe(`${window.location.origin}/api/sd-media/upload`);
        expect(uploadCall[2]).toEqual(expect.objectContaining({ headers: expect.objectContaining({ Authorization: "Bearer sk-canvas-key", "X-Video-Creation-Token-Id": "vca_account" }) }));
        expect((uploadCall[1] as FormData).get("type")).toBe("images");

        const createCall = vi.mocked(axios.post).mock.calls[1];
        expect(createCall[0]).toBe("https://gateway.example/v1/videos");
        expect(createCall[1]).toEqual({
            model: "minimax-h3-01",
            prompt: "animate this image",
            seconds: 6,
            aspect_ratio: "16:9",
            images: ["https://media.example/reference.png"],
        });
        expect(createCall[2]).toEqual(expect.objectContaining({ headers: expect.objectContaining({ Authorization: "Bearer sk-canvas-key", "X-Video-Creation-Token-Id": "vca_account" }) }));

        const pollCall = vi.mocked(axios.get).mock.calls[0];
        expect(pollCall[0]).toBe("https://gateway.example/v1/videos/video_123");
        expect(pollCall[1]).toEqual(expect.objectContaining({ headers: expect.objectContaining({ Authorization: "Bearer sk-canvas-key", "X-Video-Creation-Token-Id": "vca_account" }) }));
    });

    it("keeps polling the account that created the task after the selection changes", async () => {
        const config = {
            baseUrl: "https://gateway.example/v1",
            apiKey: "sk-canvas-key",
            apiFormat: "openai",
            model: "vca::minimax-h3-01",
            videoModel: "",
            videoSeconds: "6",
            size: "16:9",
            channels: [
                {
                    id: "vca",
                    name: "视频h3 key · 视频-h3",
                    baseUrl: "https://gateway.example/v1",
                    apiKey: "sk-canvas-key",
                    apiFormat: "openai",
                    models: [{ name: "minimax-h3-01", capability: "video", videoAccountTokenId: "vca_account", video: { durationsSeconds: [6, 10] } }],
                },
            ],
        } as AiConfig;
        vi.mocked(axios.post)
            .mockResolvedValueOnce({ data: { url: "https://media.example/reference.png" } })
            .mockImplementationOnce(async () => {
                // The user switches to another key while the job is in flight.
                config.channels = [];
                return { data: { id: "video_123" } };
            });

        await requestVideoGeneration(config, "animate this image", [{ id: "reference", name: "reference.png", type: "image/png", dataUrl: "data:image/png;base64,cmVmZXJlbmNl" }]);

        const pollCall = vi.mocked(axios.get).mock.calls[0];
        expect(pollCall[1]).toEqual(expect.objectContaining({ headers: expect.objectContaining({ "X-Video-Creation-Token-Id": "vca_account" }) }));
    });
});
