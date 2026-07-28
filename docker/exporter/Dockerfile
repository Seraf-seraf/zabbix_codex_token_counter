FROM golang:1.26.5-alpine AS build

WORKDIR /src/exporter
COPY exporter/go.mod exporter/go.sum ./
RUN go mod download
COPY exporter/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/exporter ./cmd/exporter

FROM scratch

COPY --from=build /out/exporter /exporter
USER 65532:65532
EXPOSE 4317
ENTRYPOINT ["/exporter"]
