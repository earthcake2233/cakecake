/// <reference types="vite/client" />

declare module "*.vue" {
  import type { DefineComponent } from "vue";
  const component: DefineComponent<object, object, unknown>;
  export default component;
}

interface ImportMetaEnv {
  readonly VITE_MINIBILI_API?: string;
  readonly VITE_REMOTE_API_BASE?: string;
  readonly VITE_USE_REMOTE_API?: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
