FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/api .

FROM alpine:3.22
RUN adduser -D -H -u 10001 app
COPY --from=build /out/api /usr/local/bin/api
USER app
EXPOSE 3001
HEALTHCHECK --interval=10s --timeout=3s --start-period=10s --retries=5 \
    CMD wget -q --spider http://127.0.0.1:3001/healthz || exit 1
CMD ["api"]
