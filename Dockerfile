# syntax=docker/dockerfile:1

FROM node:22-alpine AS frontend
WORKDIR /web
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories
COPY frontend-user/package.json ./
RUN npm install --registry=https://registry.npmmirror.com
COPY frontend-user/ ./
RUN npm run build

FROM golang:1.23-alpine AS backend
WORKDIR /src
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories
ENV GOPROXY=https://goproxy.cn,direct
COPY backend/go.mod ./
COPY backend/ ./
RUN go test ./...
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/gogit ./cmd/server

FROM alpine:3.21
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories \
 && apk add --no-cache ca-certificates tzdata wget su-exec \
 && mkdir -p /data/repo /app/web \
 && adduser -D -u 1001 app \
 && chown -R app:app /data /app
ENV TZ=Asia/Shanghai
WORKDIR /app
COPY --from=backend /out/gogit /app/gogit
COPY --from=frontend /web/dist /app/web
COPY docker-entrypoint.sh /app/docker-entrypoint.sh
RUN chmod +x /app/docker-entrypoint.sh
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=8s CMD wget -qO- http://127.0.0.1:8080/api/v1/health || exit 1
ENTRYPOINT ["/app/docker-entrypoint.sh"]
