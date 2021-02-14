# Server streaming call mock

Run mocking server using docker:

`docker run --rm -it -p 51000:51000 -v $(pwd):/data ghcr.io/vadimi/grpc-ditto:0.6.0-pre3 --proto /data --mocks /data`

Run mocking server using standalone app:

`grpc-ditto --proto . --mocks .`
