# Overview
`grpc-ditto` is grpc mocking server that can mock any grpc services by parsing corresponding proto file.

## Usage example:

`grpc-ditto --proto ~/dev/master-lock-svc/types --mocks jsonmocks`

this command will run a server on port `51000` by default, parse all proto files in `--proto` directory and also expose grpc reflection service.

### Mock examples

`method` is fully qualified grpc service method name
`matchesJsonPath` supports JSONPath spec: https://goessner.net/articles/JsonPath/
`equalToJson` supports protobuf specific json format: https://developers.google.com/protocol-buffers/docs/proto3#json

```
[
  {
    "request": {
      "method": "/videa.masterlock.proto.types.MasterLockService/Lock",
      "bodyPatterns": [
        {
          "matchesJsonPath": { "expression": "$.name", "equals": "lock1" }
        },
        {
          "matchesJsonPath": { "expression": "$.duration", "equals": "100" }
        }
      ]
    },
    "response": {
      "body": {
        "key": "key123"
      }
    }
  },
  {
    "request": {
      "method": "/videa.masterlock.proto.types.MasterLockService/Lock",
      "bodyPatterns": [
        {
          "equalToJson": {
            "name": "lock2",
            "duration": "200"
          }
        }
      ]
    },
    "response": {
      "body": {
        "key": "key2222222"
      }
    }
  },
  {
    "request": {
      "method": "/videa.masterlock.proto.types.MasterLockService/Lock",
      "bodyPatterns": [
        {
          "matchesJsonPath": "$.name"
        }
      ]
    },
    "response": {
      "body": {
        "key": "key_all_non_empty"
      }
    }
  }
]
````
