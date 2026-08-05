# cakecake frontend image: Vite build + Nginx hosting (proxies /api and WebSocket to backend)
# Build context is the repo root; see docker-compose.yml.
FROM node:20-alpine AS build

WORKDIR /app

ARG VITE_VIDEO_UPLOAD_DISABLED=true

COPY cakecake-vue/cakecake-web/package.json cakecake-vue/cakecake-web/package-lock.json ./
RUN npm ci

COPY cakecake-vue/cakecake-web/ ./
RUN printf 'VITE_MINIBILI_API=true\nVITE_USE_REMOTE_API=false\nVITE_VIDEO_UPLOAD_DISABLED=%s\n' \
    "$VITE_VIDEO_UPLOAD_DISABLED" > .env.production && npm run build

FROM nginx:1.27-alpine

COPY deploy/nginx-compose.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/dist /usr/share/nginx/html

EXPOSE 80
