FROM --platform=$BUILDPLATFORM golang:1.25 AS builder

WORKDIR /src

ENV CGO_ENABLED=0 GOWORK=off

ARG TARGETOS
ARG TARGETARCH

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags="-s -w" \
    -o /out/ainovel-cli \
    ./cmd/ainovel-cli

FROM alpine:3.22

RUN apk add --no-cache \
    ca-certificates \
    tzdata

# Chạy bằng user thường (UID 1000 khớp user mặc định trên đa số máy Linux,
# để file ghi ra volume không thuộc root). Cấu hình mount vào /home/ainovel/.ainovel.
RUN addgroup -g 1000 ainovel \
    && adduser -D -u 1000 -G ainovel -h /home/ainovel ainovel \
    && mkdir -p /workspace /home/ainovel/.ainovel \
    && chown ainovel:ainovel /workspace /home/ainovel/.ainovel \
    && chmod 700 /home/ainovel/.ainovel

ENV HOME=/home/ainovel

WORKDIR /workspace

COPY --from=builder /out/ainovel-cli /usr/local/bin/ainovel-cli

USER ainovel

ENTRYPOINT ["ainovel-cli"]
