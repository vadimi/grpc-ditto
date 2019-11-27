FROM registry.videa.tv/alpine:3.9

RUN apk add --no-cache ca-certificates

ARG GIT_COMMIT=unknown
ARG BUILD_NUMBER=unknown

LABEL git-commit=$GIT_COMMIT \
      build-number=$BUILD_NUMBER

RUN mkdir /app

COPY ./grpc-ditto /app/grpc-ditto

WORKDIR /app

ENTRYPOINT ["./grpc-ditto"]
