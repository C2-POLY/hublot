# syntax=docker/dockerfile:1
FROM node:22.20-alpine AS web
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM golang:1.26-alpine AS go
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web /src/web/dist ./web/dist
RUN CGO_ENABLED=0 go build -tags embedweb -trimpath -ldflags "-s -w" -o /hublot ./cmd/hublot \
 && mkdir -p /data/attachments

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=go /hublot /hublot
# A named volume mounted here inherits this ownership, so the nonroot user can write.
COPY --from=go --chown=65532:65532 /data /var/lib/hublot
EXPOSE 8080
USER nonroot
ENTRYPOINT ["/hublot"]
CMD ["serve"]
