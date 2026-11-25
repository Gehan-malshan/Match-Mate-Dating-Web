FROM golang:1.26 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/booking-api ./cmd/api && \
    CGO_ENABLED=0 go build -o /out/booking-migrate ./cmd/migrate && \
    CGO_ENABLED=0 go build -o /out/booking-expiry ./cmd/expiry-worker && \
    CGO_ENABLED=0 go build -o /out/booking-outbox ./cmd/outbox-relay && \
    CGO_ENABLED=0 go build -o /out/booking-payment-consumer ./cmd/payment-consumer
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/booking-api /booking-api
COPY --from=build /out/booking-migrate /booking-migrate
COPY --from=build /out/booking-expiry /booking-expiry
COPY --from=build /out/booking-outbox /booking-outbox
COPY --from=build /out/booking-payment-consumer /booking-payment-consumer
EXPOSE 8085
ENTRYPOINT ["/booking-api"]
