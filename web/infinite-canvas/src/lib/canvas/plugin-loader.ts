import { registerNodeDefinitions, unregisterPluginNodes } from "@/lib/canvas/node-registry";
import { getPluginRuntime } from "@/lib/canvas/plugin-runtime";
import { fetchOfficialPlugins } from "@/lib/canvas/plugin-registry";
import { canvasAssetUrl } from "@/lib/asset-url";
import { usePluginStore, type InstalledPlugin } from "@/stores/canvas/use-plugin-store";
import type { CanvasPlugin } from "@/types/canvas-plugin";
import i18n from "@/i18n";

const cleanups = new Map<string, () => void>();

// A remote plugin may export CanvasPlugin directly or a factory that receives runtime and returns CanvasPlugin.
// The factory uses runtime.React so the bundle does not need its own React copy.
async function evaluatePluginSource(source: string): Promise<CanvasPlugin> {
    const blob = new Blob([source], { type: "text/javascript" });
    const url = URL.createObjectURL(blob);
    try {
        const mod = (await import(/* @vite-ignore */ url)) as { default?: unknown; plugin?: unknown };
        const exported = mod.default ?? mod.plugin;
        const plugin = typeof exported === "function" ? (exported as (runtime: unknown) => unknown)(getPluginRuntime()) : exported;
        assertPlugin(plugin);
        return plugin;
    } finally {
        URL.revokeObjectURL(url);
    }
}

function assertPlugin(plugin: unknown): asserts plugin is CanvasPlugin {
    const value = plugin as Partial<CanvasPlugin> | null;
    if (!value || typeof value !== "object") throw new Error(i18n.t("canvas.pluginErrors.invalidExport"));
    if (!value.id || !Array.isArray(value.nodes) || !value.nodes.length) throw new Error(i18n.t("canvas.pluginErrors.missingFields"));
}

export function activatePlugin(plugin: CanvasPlugin) {
    registerNodeDefinitions(plugin.nodes, plugin.id);
    const runtime = getPluginRuntime();
    const disposers: Array<() => void> = [];
    // Inject declared styles when enabled and remove them when disabled or uninstalled.
    if (plugin.css) disposers.push(runtime.injectCSS(plugin.css, plugin.id));
    const cleanup = plugin.setup?.(runtime);
    if (typeof cleanup === "function") disposers.push(cleanup);
    if (disposers.length) cleanups.set(plugin.id, () => disposers.forEach((dispose) => dispose()));
}

export function deactivatePlugin(pluginId: string) {
    cleanups.get(pluginId)?.();
    cleanups.delete(pluginId);
    unregisterPluginNodes(pluginId);
}

async function fetchPluginSource(url: string) {
    const response = await fetch(url);
    if (!response.ok) throw new Error(i18n.t("canvas.pluginErrors.downloadFailed", { status: response.status }));
    return response.text();
}

// Add a cache-busting parameter so watch builds load the latest output.
function withCacheBust(url: string) {
    return `${url}${url.includes("?") ? "&" : "?"}t=${Date.now()}`;
}

// Install or replace a plugin from a URL and enable it immediately.
// bustCache bypasses HTTP/CDN caches during upgrades while persisting a clean URL without the timestamp query.
export async function installPluginFromUrl(url: string, opts?: { official?: boolean; bustCache?: boolean }) {
    const source = await fetchPluginSource(opts?.bustCache ? withCacheBust(url) : url);
    const plugin = await evaluatePluginSource(source);
    deactivatePlugin(plugin.id); // Replace the previous version.
    usePluginStore.getState().upsert({ id: plugin.id, name: plugin.name || plugin.id, version: plugin.version || "0.0.0", description: plugin.description, url, source, enabled: true, official: opts?.official });
    activatePlugin(plugin);
    return plugin;
}

export async function updatePlugin(record: InstalledPlugin) {
    // Upgrades must fetch the latest output and therefore always bypass caches.
    return installPluginFromUrl(record.url, { official: record.official, bustCache: true });
}

export async function setPluginEnabled(record: InstalledPlugin, enabled: boolean) {
    usePluginStore.getState().setEnabled(record.id, enabled);
    if (!enabled) {
        deactivatePlugin(record.id);
        return;
    }
    // Reload local plugins from their URL when enabled because the cached source may be stale.
    const source = record.local ? await fetchPluginSource(withCacheBust(record.url)) : record.source;
    const plugin = await evaluatePluginSource(source);
    activatePlugin(plugin);
}

export function uninstallPlugin(id: string) {
    deactivatePlugin(id);
    usePluginStore.getState().remove(id);
}

let loaded = false;

// The platform owns the plugin set. Discover every bundled/official plugin and
// enable it automatically; users should not have to maintain a second plugin
// installation state inside the canvas.
export async function ensurePluginsLoaded() {
    if (loaded) return;
    loaded = true;
    await usePluginStore.persist.rehydrate();
    await loadLocalPlugins();

    // Install all official entries in parallel. Existing records at the same
    // version keep their cached source so a transient registry/CDN outage does
    // not make already-installed nodes disappear.
    const activatedDuringInstall = await installOfficialPlugins();

    // Migrate older persisted stores that may have been written with a manual
    // enable/disable toggle. All known records are platform-managed now.
    const store = usePluginStore.getState();
    store.plugins.forEach((record) => {
        if (!record.enabled) store.setEnabled(record.id, true);
    });

    const records = usePluginStore.getState().plugins.filter((record) => record.enabled);
    await Promise.all(
        records.map(async (record) => {
            if (activatedDuringInstall.has(record.id)) return;
            try {
                // Local plugins use the latest output; other plugins use their cached source.
                const source = record.local ? await fetchPluginSource(withCacheBust(record.url)) : record.source;
                activatePlugin(await evaluatePluginSource(source));
            } catch (error) {
                console.error(`[plugin] Failed to load: ${record.id}`, error);
            }
        }),
    );
    await loadDevPlugins();
}

// Install every official registry entry without exposing an install choice in
// the UI. Returns records activated by installPluginFromUrl so the subsequent
// activation pass does not run setup twice.
async function installOfficialPlugins() {
    let entries;
    try {
        entries = await fetchOfficialPlugins();
    } catch (error) {
        // Cached plugins can still load when the public registry is unavailable.
        console.warn("[plugin] Official registry unavailable; using cached plugins", error);
        return new Set<string>();
    }

    const activated = new Set<string>();
    await Promise.all(
        entries.map(async (entry) => {
            const current = usePluginStore.getState().plugins.find((record) => record.id === entry.id);
            if (current && current.version === entry.version && current.source) {
                usePluginStore.getState().upsert({ ...current, url: entry.url, official: true, enabled: true });
                return;
            }

            try {
                await installPluginFromUrl(entry.url, { official: true, bustCache: Boolean(current) });
                activated.add(entry.id);
            } catch (error) {
                // Keep a previously cached record available for the activation
                // pass below, but do not block the rest of the plugin set.
                console.error(`[plugin] Failed to install official plugin: ${entry.id}`, error);
                if (current) usePluginStore.getState().setEnabled(current.id, true);
            }
        }),
    );
    return activated;
}

// Discover local plugins from web/public/plugins and enable them immediately.
// Refresh metadata and source for existing records while keeping the platform
// policy that every discovered plugin is active.
async function loadLocalPlugins() {
    let urls: unknown;
    try {
        const response = await fetch(canvasAssetUrl("plugins/index.json"));
        if (!response.ok) return;
        urls = await response.json();
    } catch {
        return; // Skip when no local manifest exists, such as production builds without plugins.
    }
    if (!Array.isArray(urls) || !urls.length) return;
    await Promise.all(
        urls.map(async (url: string) => {
            try {
                const source = await fetchPluginSource(withCacheBust(url));
                const plugin = await evaluatePluginSource(source);
                usePluginStore.getState().upsert({
                    id: plugin.id,
                    name: plugin.name || plugin.id,
                    version: plugin.version || "0.0.0",
                    description: plugin.description,
                    url,
                    source,
                    enabled: true,
                    local: true,
                });
            } catch (error) {
                console.error(`[plugin] Failed to discover local plugin: ${url}`, error);
            }
        }),
    );
}

// During local development, refetch VITE_DEV_PLUGINS URLs without caching or persistence on every startup.
// Together with watch builds, refreshing the page loads code changes without reinstalling the plugin.
async function loadDevPlugins() {
    const raw = import.meta.env.VITE_DEV_PLUGINS;
    if (!raw) return;
    const urls = raw
        .split(",")
        .map((item) => item.trim())
        .filter(Boolean);
    await Promise.all(
        urls.map(async (url) => {
            try {
                const source = await fetchPluginSource(withCacheBust(url));
                const plugin = await evaluatePluginSource(source);
                deactivatePlugin(plugin.id);
                activatePlugin(plugin);
                console.info(`[plugin] Dev plugin loaded: ${plugin.id} (${url})`);
            } catch (error) {
                console.error(`[plugin] Failed to load dev plugin: ${url}`, error);
            }
        }),
    );
}
