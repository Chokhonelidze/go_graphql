FROM golang:latest AS build
ENV CGO_ENABLED=0 GO111MODULE=on
WORKDIR /build

# Copy module files and download deps
COPY app/go.mod app/go.sum ./
RUN go mod download

# Copy application sources into /build (not /build/app)
COPY app/ ./
RUN go run github.com/99designs/gqlgen generate
RUN go build -o /server .

FROM alpine:latest AS final
WORKDIR /app
COPY --from=build /server /app/server
EXPOSE 8080
ENTRYPOINT ["/app/server"]
