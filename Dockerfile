FROM golang:1.27.1-alpine AS build
WORKDIR /build
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 go build -buildvcs=false -trimpath -ldflags="-s -w" -o /interview-tracker .
FROM alpine:3.23
RUN apk add --no-cache ca-certificates tzdata && adduser -D app
WORKDIR /app
COPY --from=build /interview-tracker ./interview-tracker
COPY web/ ./web/
USER app
ENV LISTEN_ADDR=0.0.0.0:8091 WEB_DIR=/app/web GOMEMLIMIT=64MiB
EXPOSE 8091
ENTRYPOINT ["/app/interview-tracker"]
