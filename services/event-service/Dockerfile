FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/event-api ./cmd/api && CGO_ENABLED=0 go build -o /out/event-migrate ./cmd/migrate && CGO_ENABLED=0 go build -o /out/event-outbox-relay ./cmd/outbox-relay && CGO_ENABLED=0 go build -o /out/event-seed ./cmd/seed-dev
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/event-api /event-api
COPY --from=build /out/event-migrate /event-migrate
COPY --from=build /out/event-outbox-relay /event-outbox-relay
COPY --from=build /out/event-seed /event-seed
EXPOSE 8082
ENTRYPOINT ["/event-api"]
