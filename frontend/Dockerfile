# ── Stage 1: build the Vite SPA ──────────────────────────────────────────────
FROM node:20-alpine AS builder

WORKDIR /build

# Cache deps first.
COPY package.json package-lock.json* ./
RUN npm ci

COPY . .

# VITE_API_BASE_URL is a relative path so the browser asks nginx for /api/v1/...
# and nginx (in the runtime stage) proxies to the backend container.
# This is a build-time constant — it's baked into the JS bundle.
ENV VITE_API_BASE_URL=/api/v1
RUN npm run build

# ── Stage 2: nginx serves the built SPA + proxies /api to the backend ────────
FROM nginx:1.27-alpine

# Replace the default nginx config with our SPA + reverse-proxy config.
RUN rm /etc/nginx/conf.d/default.conf
COPY nginx.conf /etc/nginx/conf.d/learnthor.conf

# Copy the built static assets.
COPY --from=builder /build/dist /usr/share/nginx/html

EXPOSE 80

# nginx foreground process (container stays alive until stopped).
CMD ["nginx", "-g", "daemon off;"]
