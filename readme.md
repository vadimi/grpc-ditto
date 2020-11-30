# Overview
`grpc-ditto` is grpc mocking server that can mock any grpc services by parsing corresponding proto file.

## Usage example:

`grpc-ditto --proto myprotodir --mocks jsonmocksdir`

this command will run a server on port `51000` by default, parse all proto files in `--proto` directory, load all mocks from json files in `--mocks` directory and also expose grpc reflection service.

### Mock format

- `method` is fully qualified grpc service method name
- `matchesJsonPath` supports JSONPath spec: https://goessner.net/articles/JsonPath/
- `equalToJson` supports protobuf specific json format: https://developers.google.com/protocol-buffers/docs/proto3#json
- multiple `bodyPatterns` should all match in order for a request to match

```json
[
  {
    "request": {
      "method": "/greet.Greeter/SayHello",
      "bodyPatterns": [
        {
          "matchesJsonPath": {"expression": "$.name", "eq": "Bob"}
        }
      ]
    },
    "response": {
      "body": {"message": "hello Bob"}
    }
  },
  {
    "request": {
      "method": "/greet.Greeter/SayHello",
      "bodyPatterns": [
        {
          "matchesJsonPath": "$.name"
        }
      ]
    },
    "response": {
      "body": {"message": "hello human"}
    }
  }
]
````
