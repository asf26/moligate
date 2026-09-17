/**
 * Reference video / audio the workspace uploads to the gateway before creating
 * a video task. The gateway stores the file with the account's own credential
 * and returns a public URL, which is what the model then receives.
 */
export type VideoReferenceMedia = {
    id: string;
    name: string;
    kind: "video" | "audio";
    mimeType: string;
    /** Local (blob or asset) URL used for preview and for reading the bytes back. */
    url: string;
    storageKey?: string;
};

/** "references" is the multi-reference workflow, "first_last_frame" submits workflow_id=fl2v. */
export type VideoOperationMode = "references" | "first_last_frame";

export function normalizeVideoOperationMode(value: string | undefined): VideoOperationMode {
    return value === "first_last_frame" ? "first_last_frame" : "references";
}

/** Anything the canvas can hand over as a reference clip or track. */
type ReferenceMediaSource = {
    id: string;
    name: string;
    /** MIME type as stored on the node. */
    type: string;
    url: string;
    storageKey?: string;
};

/**
 * Adapts canvas reference nodes (which use their own media shape) to the media
 * a video request forwards. Without this the clips and tracks a user connects
 * on the canvas are collected but never sent.
 */
export function toVideoReferenceMedia(items: ReferenceMediaSource[] | undefined, kind: "video" | "audio"): VideoReferenceMedia[] {
    return (items || [])
        .filter((item) => Boolean(item.url || item.storageKey))
        .map((item) => ({ id: item.id, name: item.name, kind, mimeType: item.type, url: item.url, storageKey: item.storageKey }));
}
