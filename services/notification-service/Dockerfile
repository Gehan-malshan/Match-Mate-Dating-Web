FROM golang:1.26 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/notification-api ./cmd/api && \
    CGO_ENABLED=0 go build -o /out/notification-migrate ./cmd/migrate && \
    CGO_ENABLED=0 go build -o /out/notification-consumer ./cmd/event-consumer && \
    CGO_ENABLED=0 go build -o /out/notification-worker ./cmd/delivery-worker && \
    CGO_ENABLED=0 go build -o /out/notification-delivery-replay ./cmd/delivery-replay && \
    CGO_ENABLED=0 go build -o /out/notification-provider-preflight ./cmd/provider-preflight

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/notification-api /notification-api
COPY --from=build /out/notification-migrate /notification-migrate
COPY --from=build /out/notification-consumer /notification-consumer
COPY --from=build /out/notification-worker /notification-worker
COPY --from=build /out/notification-delivery-replay /notification-delivery-replay
COPY --from=build /out/notification-provider-preflight /notification-provider-preflight
EXPOSE 8086
ENTRYPOINT ["/notification-api"]

