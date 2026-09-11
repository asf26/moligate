import { LoaderCircle } from "lucide-react";
import { useTranslation } from "react-i18next";

export function CanvasRefreshShell() {
    const { t } = useTranslation();

    return (
        <main className="relative grid h-full min-h-0 place-items-center overflow-hidden bg-background text-foreground" aria-live="polite">
            <div
                className="absolute inset-0 opacity-60"
                style={{
                    backgroundImage: "radial-gradient(circle, var(--border) 1px, transparent 1px)",
                    backgroundSize: "28px 28px",
                }}
            />

            <section className="relative flex w-[min(22rem,calc(100%-2rem))] flex-col items-center rounded-3xl border border-blue-100 bg-white/90 px-8 py-9 text-center shadow-[0_24px_70px_-36px_rgba(37,99,235,0.55)] backdrop-blur-xl">
                <div className="relative mb-5 grid size-16 place-items-center rounded-2xl border border-blue-100 bg-blue-50 text-blue-600 shadow-inner">
                    <span className="absolute inset-0 animate-ping rounded-2xl border border-blue-300/60" />
                    <LoaderCircle aria-hidden="true" className="relative size-7 animate-spin" />
                </div>
                <p className="text-sm font-semibold tracking-wide text-slate-800">{t("canvas.loading")}</p>
                <div className="mt-6 h-1.5 w-44 overflow-hidden rounded-full bg-blue-50">
                    <div className="h-full w-1/2 animate-[canvas-loading_1.6s_ease-in-out_infinite] rounded-full bg-gradient-to-r from-blue-500 to-cyan-400" />
                </div>
            </section>
            <style>{`@keyframes canvas-loading { 0%, 100% { transform: translateX(-100%); } 50% { transform: translateX(200%); } }`}</style>
        </main>
    );
}
