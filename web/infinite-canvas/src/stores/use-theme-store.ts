import { create } from "zustand";
import { persist } from "zustand/middleware";

export type ThemeName = "light" | "dark";

type ThemeStore = {
    theme: ThemeName;
    setTheme: (theme: ThemeName) => void;
};

export const useThemeStore = create<ThemeStore>()(
    persist(
        (set) => ({
            theme: "light",
            setTheme: () => set({ theme: "light" }),
        }),
        {
            name: "infinite-canvas:theme_store",
            merge: (_persistedState, currentState) => currentState,
        },
    ),
);
