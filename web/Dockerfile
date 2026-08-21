# syntax=docker/dockerfile:1.7

FROM node:24-alpine AS build

WORKDIR /src

COPY web/package.json web/package-lock.json ./
RUN --mount=type=cache,target=/root/.npm npm ci

COPY web/ ./
RUN npm run build

FROM nginx:1.28-alpine

ARG VCS_REF=unknown
ARG SOURCE_URL=https://github.com/cuihe500/astro

LABEL org.opencontainers.image.title="Astro Web" \
      org.opencontainers.image.description="Astro 容器即服务平台 Web 界面" \
      org.opencontainers.image.source=$SOURCE_URL \
      org.opencontainers.image.revision=$VCS_REF

COPY web/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /src/dist/ /usr/share/nginx/html/

EXPOSE 80

HEALTHCHECK --interval=10s --timeout=5s --start-period=15s --retries=6 \
    CMD wget -q -T 3 -O /dev/null http://127.0.0.1/ || exit 1
