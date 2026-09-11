import { createBrowserRouter, Outlet } from "react-router-dom";

import { AnalyticsTracker } from "@/components/layout/analytics-tracker";
import CanvasEntryPage from "@/components/canvas/canvas-entry-page";
import CanvasProjectPage from "@/pages/canvas/project";
import CanvasPage from "@/pages/canvas";
import AssetsPage from "@/pages/assets";
import ConfigPage from "@/pages/config";
import ImagePage from "@/pages/image";
import NotFound from "@/pages/not-found";
import PromptsPage from "@/pages/prompts";
import UserLayout from "@/layouts/user-layout";
import VideoPage from "@/pages/video";

export const router = createBrowserRouter(
    [
        {
            element: (
                <UserLayout>
                    <AnalyticsTracker />
                    <Outlet />
                </UserLayout>
            ),
            children: [
                { index: true, element: <CanvasEntryPage /> },
                // Keep the workbenches available from the canvas app navigation.
                // They stay inside this iframe and never replace the dashboard shell.
                { path: "image", element: <ImagePage /> },
                { path: "video", element: <VideoPage /> },
                { path: "prompts", element: <PromptsPage /> },
                { path: "assets", element: <AssetsPage /> },
                { path: "config", element: <ConfigPage /> },
                { path: "canvas", element: <CanvasPage /> },
                // The upstream route shape is kept as a compatibility alias for
                // imported links, while the integrated app still uses /:id.
                { path: "canvas/:id", element: <CanvasProjectPage /> },
                { path: ":id", element: <CanvasProjectPage /> },
            ],
        },
        { path: "*", element: <NotFound /> },
    ],
    { basename: "/canvas-app" },
);
