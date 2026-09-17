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
        // Mirrors the real resolver: capabilities belong to the resolved model.
        resolveModelVideoMetadata: (config: AiConfig, value: string) => {
            const model = modelNameOf(value);
            return channelFor(config, value)?.models.find((item) => item.name === model)?.video;
        },
    };
});

vi.mock("../model-plugin", () => ({ runModelPlugin: vi.fn() }));

describe("requestVideoGeneration", () => {
    beforeEach(() => {
        // Reset, not just clear: a queued mockResolvedValueOnce would otherwise
        // leak into the next test and answer the wrong request.
        vi.mocked(axios.post).mockReset();
        vi.mocked(axios.get).mockReset();
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
            // A lone reference image stays on the single-reference field, which is
            // what CTMOAI's own console sends for that case.
            input_reference: "https://media.example/reference.png",
            prompt_enhance: true,
        });
        expect(createCall[2]).toEqual(expect.objectContaining({ headers: expect.objectContaining({ Authorization: "Bearer sk-canvas-key", "X-Video-Creation-Token-Id": "vca_account" }) }));

        const pollCall = vi.mocked(axios.get).mock.calls[0];
        expect(pollCall[0]).toBe("https://gateway.example/v1/videos/video_123");
        expect(pollCall[1]).toEqual(expect.objectContaining({ headers: expect.objectContaining({ Authorization: "Bearer sk-canvas-key", "X-Video-Creation-Token-Id": "vca_account" }) }));
    });

    it("submits the size the model maps to the selected ratio", async () => {
        vi.mocked(axios.post)
            .mockResolvedValueOnce({ data: { url: "https://media.example/reference.png" } })
            .mockResolvedValueOnce({ data: { id: "video_123" } });
        const config = dedicatedAccountConfig({
            name: "minimax-h3-original-768p",
            videoAccountTokenId: "vca_account",
            video: { resolution: "768p", durationsSeconds: [4, 10], ratios: ["16:9", "9:16"], ratioSizes: { "16:9": "1376x768", "9:16": "768x1376" }, maxImages: 9 },
        });

        await requestVideoGeneration({ ...config, size: "9:16" } as AiConfig, "animate", [referenceImage()]);

        expect(vi.mocked(axios.post).mock.calls[1][1]).toEqual({
            model: "minimax-h3-original-768p",
            prompt: "animate",
            seconds: 4,
            aspect_ratio: "9:16",
            size: "768x1376",
            input_reference: "https://media.example/reference.png",
            prompt_enhance: true,
        });
    });

    it("sends workflow_id=fl2v and at most two images in first/last frame mode", async () => {
        // Three references are supplied, but the mode only uploads the two frames.
        vi.mocked(axios.post)
            .mockResolvedValueOnce({ data: { url: "https://media.example/frame.png" } })
            .mockResolvedValueOnce({ data: { url: "https://media.example/frame.png" } })
            .mockResolvedValueOnce({ data: { id: "video_123" } });
        const config = dedicatedAccountConfig({
            name: "minimax-h3-original-768p",
            videoAccountTokenId: "vca_account",
            video: { resolution: "768p", durationsSeconds: [4], ratios: ["16:9"], ratioSizes: { "16:9": "1376x768" }, maxImages: 9, supportsFirstLastFrame: true },
        });

        await requestVideoGeneration({ ...config, size: "16:9", videoOperationMode: "first_last_frame" } as AiConfig, "transition", [referenceImage(), referenceImage(), referenceImage()]);

        const body = vi.mocked(axios.post).mock.calls.at(-1)?.[1] as Record<string, unknown>;
        expect(body.workflow_id).toBe("fl2v");
        // CTMOAI's console names the same workflow with mode, so send both.
        expect(body.mode).toBe("first_last_frame");
        expect(body.prompt_enhance).toBe(true);
        expect((body.images as string[]).length).toBe(2);
        expect(body.input_reference).toBeUndefined();
        expect(body.reference_videos).toBeUndefined();
    });

    it("uploads and forwards reference videos and audios when the model allows them", async () => {
        vi.mocked(axios.post)
            .mockResolvedValueOnce({ data: { images: ["https://media.example/image.png"] } })
            .mockResolvedValueOnce({ data: { videos: ["https://media.example/clip.mp4"] } })
            .mockResolvedValueOnce({ data: { audios: ["https://media.example/track.mp3"] } })
            .mockResolvedValueOnce({ data: { id: "video_123" } });
        const config = dedicatedAccountConfig({
            name: "sd-2-vip-480",
            videoAccountTokenId: "vca_account",
            video: { resolution: "480p", durationsSeconds: [5, 6], ratios: ["9:16", "16:9"], maxImages: 9, maxVideos: 3, maxAudios: 3 },
        });

        await requestVideoGeneration({ ...config, size: "16:9" } as AiConfig, "blend these", [referenceImage()], {
            referenceVideos: [{ id: "v1", name: "clip.mp4", kind: "video", mimeType: "video/mp4", url: "data:video/mp4;base64,Y2xpcA==" }],
            referenceAudios: [{ id: "a1", name: "track.mp3", kind: "audio", mimeType: "audio/mpeg", url: "data:audio/mpeg;base64,dHJhY2s=" }],
        });

        const uploads = vi.mocked(axios.post).mock.calls.filter((call) => call[1] instanceof FormData).map((call) => (call[1] as FormData).get("type"));
        expect(uploads).toEqual(["images", "videos", "audios"]);

        const body = vi.mocked(axios.post).mock.calls.at(-1)?.[1] as Record<string, unknown>;
        expect(body).toEqual({
            model: "sd-2-vip-480",
            prompt: "blend these",
            seconds: 6,
            aspect_ratio: "16:9",
            // Mixing material keeps the array form even with a single image.
            images: ["https://media.example/image.png"],
            reference_videos: ["https://media.example/clip.mp4"],
            reference_audios: ["https://media.example/track.mp3"],
            prompt_enhance: true,
        });
        // Seedance takes no size and has no first/last frame workflow.
        expect(body.size).toBeUndefined();
        expect(body.workflow_id).toBeUndefined();
    });

    it("omits reference kinds the model does not advertise", async () => {
        vi.mocked(axios.post)
            .mockResolvedValueOnce({ data: { images: ["https://media.example/image.png"] } })
            .mockResolvedValueOnce({ data: { id: "video_123" } });
        const config = dedicatedAccountConfig({
            name: "minimax-h3-quantized-768p",
            videoAccountTokenId: "vca_account",
            video: { resolution: "768p", durationsSeconds: [4], ratios: ["16:9"], maxImages: 4, maxVideos: 0, maxAudios: 0 },
        });

        await requestVideoGeneration({ ...config, size: "16:9" } as AiConfig, "animate", [referenceImage()], {
            referenceVideos: [{ id: "v1", name: "clip.mp4", kind: "video", mimeType: "video/mp4", url: "data:video/mp4;base64,Y2xpcA==" }],
        });

        const uploads = vi.mocked(axios.post).mock.calls.filter((call) => call[1] instanceof FormData).map((call) => (call[1] as FormData).get("type"));
        expect(uploads).toEqual(["images"]);
        expect((vi.mocked(axios.post).mock.calls.at(-1)?.[1] as Record<string, unknown>).reference_videos).toBeUndefined();
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

function referenceImage() {
    return { id: "reference", name: "reference.png", type: "image/png", dataUrl: "data:image/png;base64,cmVmZXJlbmNl" };
}

/** A config exposing one dedicated-account model, matching the canvas config shape. */
function dedicatedAccountConfig(model: { name: string; videoAccountTokenId: string; video: NonNullable<AiConfig["channels"][number]["models"][number]["video"]> }) {
    return {
        baseUrl: "https://gateway.example/v1",
        apiKey: "sk-canvas-key",
        apiFormat: "openai",
        model: `vca::${model.name}`,
        videoModel: "",
        videoSeconds: "6",
        size: "16:9",
        channels: [{ id: "vca", name: "key · group", baseUrl: "https://gateway.example/v1", apiKey: "sk-canvas-key", apiFormat: "openai", models: [{ ...model, capability: "video" }] }],
    };
}
