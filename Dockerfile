# ---------- 构建阶段 ----------
FROM golang:1.22-alpine AS build
ARG VERSION=dev
WORKDIR /src
COPY go.mod ./
COPY *.go ./
RUN CGO_ENABLED=0 go build -trimpath \
    -ldflags="-s -w -X main.version=${VERSION}" \
    -o /out/bing-search-api .

# ---------- 运行阶段 ----------
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
COPY --from=build /out/bing-search-api /usr/local/bin/bing-search-api
ENV PORT=8080
EXPOSE 8080
ENTRYPOINT ["bing-search-api"]
