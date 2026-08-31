FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/matchmaking-api ./cmd/api && \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/matchmaking-migrate ./cmd/migrate && \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/matchmaking-seed-dev ./cmd/seed-dev

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/matchmaking-api /matchmaking-api
COPY --from=build /out/matchmaking-migrate /matchmaking-migrate
COPY --from=build /out/matchmaking-seed-dev /matchmaking-seed-dev
EXPOSE 8083
ENTRYPOINT ["/matchmaking-api"]
