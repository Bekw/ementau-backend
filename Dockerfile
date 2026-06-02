FROM golang:1.22-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /app/server ./cmd/api

FROM alpine:latest
WORKDIR /app
COPY --from=build /app/server .
RUN mkdir -p /app/uploads
EXPOSE 8080
CMD ["./server"]