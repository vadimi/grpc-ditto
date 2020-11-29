FROM alpine:3.12

RUN apk add --no-cache ca-certificates

RUN mkdir /app

COPY ./grpc-ditto /app/grpc-ditto

WORKDIR /app

ENTRYPOINT ["./grpc-ditto"]
