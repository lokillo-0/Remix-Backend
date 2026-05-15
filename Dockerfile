FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

COPY databases/ ./databases/
COPY .env ./

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o xenon .

FROM alpine:latest

RUN apk --no-cache add bash ca-certificates tzdata

WORKDIR /app

RUN mkdir -p logs databases

COPY --from=builder /app/xenon .

COPY docker-entrypoint.sh .
RUN chmod +x docker-entrypoint.sh

EXPOSE 80 443

ENTRYPOINT ["./docker-entrypoint.sh"]