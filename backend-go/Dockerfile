FROM --platform=$BUILDPLATFORM golang:1.24-alpine AS build
ARG TARGETOS=linux
ARG TARGETARCH
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath -ldflags="-s -w" -o /out/elvove-api ./cmd/api

FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata libwebp-tools && adduser -D -u 10001 app
WORKDIR /app
COPY --from=build /out/elvove-api /usr/local/bin/elvove-api
RUN mkdir -p /app/uploads && chown -R app:app /app
USER app
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/elvove-api"]
