FROM golang:1.23.1 as builder

WORKDIR  /app

COPY .  .

RUN  CGO_ENABLED=0 go build  -o  ingress-manager main.go



FROM   alpine:3.20.3

WORKDIR /app

COPY --from=builder /app/ingress-manager   .

CMD ["./ingress-manager"]