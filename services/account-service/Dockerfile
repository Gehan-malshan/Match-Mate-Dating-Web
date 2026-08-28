FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/account-api ./cmd/api && \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/account-migrate ./cmd/migrate && \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/account-seed-dev ./cmd/seed-dev && \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/account-outbox ./cmd/outbox-relay

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/account-api /account-api
COPY --from=build /out/account-migrate /account-migrate
COPY --from=build /out/account-seed-dev /account-seed-dev
COPY --from=build /out/account-outbox /account-outbox
EXPOSE 8081
USER nonroot:nonroot
ENTRYPOINT ["/account-api"]
