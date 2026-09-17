import { describe, expect, it } from "vitest";

import { normalizeVideoOperationMode, toVideoReferenceMedia } from "../video";

describe("normalizeVideoOperationMode", () => {
    it("defaults to the multi-reference workflow", () => {
        expect(normalizeVideoOperationMode(undefined)).toBe("references");
        expect(normalizeVideoOperationMode("")).toBe("references");
        expect(normalizeVideoOperationMode("first_last_frame")).toBe("first_last_frame");
    });
});

describe("toVideoReferenceMedia", () => {
    it("keeps the canvas media a video request needs to upload", () => {
        expect(
            toVideoReferenceMedia(
                [
                    { id: "v1", name: "clip.mp4", type: "video/mp4", url: "blob:clip", storageKey: "media_v1" },
                    { id: "v2", name: "empty.mp4", type: "video/mp4", url: "" },
                ],
                "video",
            ),
        ).toEqual([{ id: "v1", name: "clip.mp4", kind: "video", mimeType: "video/mp4", url: "blob:clip", storageKey: "media_v1" }]);
    });

    it("accepts a stored asset that has no in-memory url", () => {
        expect(toVideoReferenceMedia([{ id: "a1", name: "track.mp3", type: "audio/mpeg", url: "", storageKey: "media_a1" }], "audio")).toEqual([
            { id: "a1", name: "track.mp3", kind: "audio", mimeType: "audio/mpeg", url: "", storageKey: "media_a1" },
        ]);
    });

    it("returns nothing for a node with no reference media", () => {
        expect(toVideoReferenceMedia(undefined, "video")).toEqual([]);
    });
});
