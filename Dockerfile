FROM alpine:3.14

RUN apk add --no-cache ca-certificates

RUN mkdir /app

COPY ./grpc-ditto /app/grpc-ditto

WORKDIR /app

ENTRYPOINT ["./grpc-ditto"]
