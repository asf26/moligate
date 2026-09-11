/** Resolve a bundled canvas asset under the Vite base path. */
export function canvasAssetUrl(path: string) {
    const base = import.meta.env.BASE_URL || "/";
    return `${base.replace(/\/+$/, "")}/${path.replace(/^\/+/, "")}`;
}
