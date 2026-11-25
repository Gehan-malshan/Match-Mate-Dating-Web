FROM golang:1.26 AS build
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/payment-api ./cmd/api && \
    CGO_ENABLED=0 go build -o /out/payment-migrate ./cmd/migrate && \
    CGO_ENABLED=0 go build -o /out/payment-outbox ./cmd/outbox-relay && \
    CGO_ENABLED=0 go build -o /out/payment-reconciliation ./cmd/reconciliation-worker

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/payment-api /payment-api
COPY --from=build /out/payment-migrate /payment-migrate
COPY --from=build /out/payment-outbox /payment-outbox
COPY --from=build /out/payment-reconciliation /payment-reconciliation
EXPOSE 8084
ENTRYPOINT ["/payment-api"]
