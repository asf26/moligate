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

vi.mock("@/stores/use-config-store", () => ({
    boolConfig: vi.fn(),
    buildApiUrl: (baseUrl: string, path: string) => `${baseUrl}${path}`,
    modelOptionName: (model: string) => model.split("::").at(-1) || model,
    resolveModelRequestConfig: (config: AiConfig) => config,
    resolveModelScript: () => "",
}));

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
});
