import { vi } from "vitest";

// Mock image imports (Vite resolves to URL strings)
vi.mock("@/assets/akari.jpg", () => ({ default: "/mock/akari.jpg" }));
vi.mock("@/assets/loading.gif", () => ({ default: "/mock/loading.gif" }));
vi.mock("@/assets/personal_space/latertowatch.png", () => ({ default: "/mock/latertowatch.png" }));
vi.mock("@/assets/square_01.jpg", () => ({ default: "/mock/square_01.jpg" }));
vi.mock("@/assets/square_02.jpg", () => ({ default: "/mock/square_02.jpg" }));
vi.mock("@/assets/live_01.png", () => ({ default: "/mock/live_01.png" }));
vi.mock("@/assets/live_02.png", () => ({ default: "/mock/live_02.png" }));
vi.mock("@/styles/cm-del-msgbox.scss", () => ({}));
vi.mock("@/style/mixin", () => ({}));
