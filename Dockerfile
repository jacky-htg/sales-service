FROM golang:alpine

WORKDIR /app
COPY sales-service /app/
COPY ./cmd/cmd-sales-service /app/

CMD ["/app/sales-service"]
