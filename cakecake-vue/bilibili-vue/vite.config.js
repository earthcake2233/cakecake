/// <reference types="vitest" />
import { defineConfig, loadEnv } from "vite";
import vue from "@vitejs/plugin-vue";
import path from "path";
import { fileURLToPath } from "url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, __dirname, "");
  const minibili =
    env.VITE_MINIBILI_API === "true" || env.VITE_MINIBILI_API === "1";
  const backend = (env.VITE_REMOTE_API_BASE || "http://127.0.0.1:8080").replace(
    /\/$/,
    ""
  );

  const proxy = minibili
    ? {
        /** 大体积 multipart 上传经代理时易超时，拉长读写超时（与 axios mbUploadVideo 600s 对齐） */
        "/api/v1": {
          target: backend,
          changeOrigin: true,
          ws: true,
          timeout: 600000,
          proxyTimeout: 600000,
          configure(proxy) {
            /** 开发态 WS（弹幕/聊天）在后端重启、HMR 刷新时断开，属正常现象，避免刷屏 */
            const benignWsCodes = new Set([
              "ECONNABORTED",
              "ECONNRESET",
              "ECONNREFUSED",
              "EPIPE"
            ]);
            const finishProxyError = (res, statusCode = 502) => {
              if (!res || res.headersSent) return;
              if (typeof res.writeHead === "function") {
                res.writeHead(statusCode, { "Content-Type": "application/json" });
                res.end(
                  JSON.stringify({
                    code: statusCode,
                    msg: "后端未启动或不可达，请确认 127.0.0.1:8080 已运行"
                  })
                );
                return;
              }
              if (typeof res.destroy === "function") {
                res.destroy();
              }
            };
            proxy.on("error", (err, _req, res) => {
              if (err && benignWsCodes.has(err.code)) {
                finishProxyError(res);
                return;
              }
              console.error("[vite proxy]", err);
            });
            proxy.on("proxyReqWs", (_proxyReq, _req, socket) => {
              socket.on("error", err => {
                if (err && benignWsCodes.has(err.code)) return;
              });
            });
            proxy.on("proxyReq", (proxyReq, req) => {
              if (proxyReq.getHeader("x-forwarded-for")) return;
              const raw = req.socket?.remoteAddress || "";
              const ip = raw.replace(/^::ffff:/, "");
              if (ip) proxyReq.setHeader("X-Forwarded-For", ip);
            });
          }
        }
      }
    : {
        "/api": {
          target: backend || "http://127.0.0.1:8080",
          changeOrigin: true,
          rewrite: p => p.replace(/^\/api/, "/api/v1")
        }
      };

  return {
    plugins: [vue()],
    base: "./",
    resolve: {
      extensions: [".vue", ".mjs", ".js", ".mts", ".ts", ".jsx", ".tsx", ".json"],
      alias: {
        "@": path.resolve(__dirname, "src")
      }
    },
    server: {
      port: 8888,
      open: true,
      proxy
    },
    test: {
      environment: "jsdom", testTimeout: 30000,
      globals: true,
      include: ["src/**/*.{test,spec}.{js,ts,jsx,tsx}"],
      setupFiles: ["vitest.setup.js"]
    },
    build: {
      outDir: "dist",
      assetsDir: "static",
      sourcemap: false
    },
    css: {
      preprocessorOptions: {
        scss: {
          silenceDeprecations: ["legacy-js-api", "import"]
        }
      }
    }
  };
});
