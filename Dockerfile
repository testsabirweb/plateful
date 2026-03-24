# Multi-stage build: shared compile, separate minimal images for api and worker.
# Build: docker build --target api -t plateful-api .
#        docker build --target worker -t plateful-worker .

FROM golang:1.25-alpine AS build
WORKDIR /src

RUN apk add --no-cache ca-certificates git

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api \
 && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/worker ./cmd/worker

FROM alpine:3.21 AS api
RUN apk add --no-cache ca-certificates tzdata \
 && adduser -D -H -u 65532 nonroot
COPY --from=build /out/api /plateful-api
USER nonroot
EXPOSE 8080
ENTRYPOINT ["/plateful-api"]

FROM alpine:3.21 AS worker
RUN apk add --no-cache ca-certificates tzdata \
 && adduser -D -H -u 65532 nonroot
COPY --from=build /out/worker /plateful-worker
USER nonroot
ENTRYPOINT ["/plateful-worker"]
