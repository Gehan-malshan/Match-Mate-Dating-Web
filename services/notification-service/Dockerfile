FROM golang:1.26 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/notification-api ./cmd/api && \
    CGO_ENABLED=0 go build -o /out/notification-migrate ./cmd/migrate && \
    CGO_ENABLED=0 go build -o /out/notification-consumer ./cmd/event-consumer && \
    CGO_ENABLED=0 go build -o /out/notification-worker ./cmd/delivery-worker

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/notification-api /notification-api
COPY --from=build /out/notification-migrate /notification-migrate
COPY --from=build /out/notification-consumer /notification-consumer
COPY --from=build /out/notification-worker /notification-worker
EXPOSE 8086
ENTRYPOINT ["/notification-api"]

