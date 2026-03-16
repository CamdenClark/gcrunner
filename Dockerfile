FROM golang:1.26-alpine AS build
WORKDIR /app
COPY function/go.mod function/go.sum ./function/
RUN cd function && go mod download
COPY function/ ./function/
RUN cd function && CGO_ENABLED=0 go build -o /server ./cmd/server

FROM gcr.io/distroless/static:nonroot
COPY --from=build /server /server
ENTRYPOINT ["/server"]
