FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/moderation-api ./cmd/api && \
    CGO_ENABLED=0 go build -o /out/moderation-migrate ./cmd/migrate && \
    CGO_ENABLED=0 go build -o /out/moderation-outbox-relay ./cmd/outbox-relay && \
    CGO_ENABLED=0 go build -o /out/moderation-expiry-worker ./cmd/expiry-worker
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/moderation-api /moderation-api
COPY --from=build /out/moderation-migrate /moderation-migrate
COPY --from=build /out/moderation-outbox-relay /moderation-outbox-relay
COPY --from=build /out/moderation-expiry-worker /moderation-expiry-worker
EXPOSE 8087
ENTRYPOINT ["/moderation-api"]
