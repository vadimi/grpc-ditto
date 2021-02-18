# Bidi streaming call mock

Run mocking server using docker:

`docker run --rm -it -p 51000:51000 -v $(pwd):/data ghcr.io/vadimi/grpc-ditto --proto /data --mocks /data/mocks.json`

Run mocking server using standalone app:

`grpc-ditto --proto . --mocks mocks.json`

yaml is also supported:

`grpc-ditto --proto . --mocks mocks.yaml`
