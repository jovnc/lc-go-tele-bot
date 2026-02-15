# syntax=docker/dockerfile:1

FROM golang:1.22-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY main.go ./
COPY internal ./internal

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/bot .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/bot /bot

EXPOSE 8080
ENTRYPOINT ["/bot"]
