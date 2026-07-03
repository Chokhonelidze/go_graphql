
FROM golang:latest AS build
ENV CGO_ENABLED=0 GO111MODULE=on
ENV GOPROXY=direct
ENV GOSUMDB=off

WORKDIR /build

# Copy module files and download deps
COPY ./go.mod ./go.sum ./
RUN go mod download

# Copy application sources into /build (not /build/app)
COPY ./ ./
RUN go run github.com/99designs/gqlgen generate
RUN go build -o /server .


FROM alpine  AS final

# ARG AZURE_CLIENT_ID
# ARG AZURE_CLIENT_SECRET
# ARG AZURE_TENANT_ID
# ENV AZURE_CLIENT_ID=${AZURE_CLIENT_ID}
# ENV AZURE_CLIENT_SECRET=${AZURE_CLIENT_SECRET}
# ENV AZURE_TENANT_ID=${AZURE_TENANT_ID}

# RUN apt-get update
# RUN apt-get install azure-cli -y
# RUN curl -sSfL https://atlasgo.sh | sh -s -- -y
# RUN az login --service-principal -u $AZURE_CLIENT_ID -p $AZURE_CLIENT_SECRET --tenant $AZURE_TENANT_ID
WORKDIR /app
COPY --from=build /server /server
EXPOSE 8080
ENTRYPOINT ["/server"]
