package service_test

import data.testservice

envoy_input := {
  "attributes": {
    "destination": {
      "address": {
        "socketAddress": {
          "address": "172.22.0.4",
          "portValue": 8080
        }
      }
    },
    "metadataContext": {},
    "request": {
      "http": {
        "headers": {
          ":authority": "localhost:8080",
          ":method": "GET",
          ":path": "/testservice/get",
          ":scheme": "http",
          "accept": "*/*",
          "user-agent": "curl/8.7.1",
          "x-forwarded-proto": "http",
          "x-request-id": "db623bd1-503b-4319-b2f7-6438e0acc2dc"
        },
        "host": "localhost:8080",
        "id": "15977865728716054527",
        "method": "GET",
        "path": "/testservice/get",
        "protocol": "HTTP/1.1",
        "scheme": "http"
      },
      "time": {
        "nanos": 898459000,
        "seconds": 1757944467
      }
    },
    "routeMetadataContext": {},
    "source": {
      "address": {
        "socketAddress": {
          "address": "140.82.121.5",
          "portValue": 18328
        }
      }
    }
  },
  "parsed_body": null,
  "parsed_path": [
    "get"
  ],
  "parsed_query": {},
  "truncated_body": false,
  "version": {
    "encoding": "protojson",
    "ext_authz": "v3"
  },
  "time": "2025-09-15T13:54:27Z"
}

test_path if {
    testservice.path == "/get" with input as envoy_input
}

test_method if {
    testservice.method == "get" with input as envoy_input
}

test_global_rule if {
    testservice.allow_request with input as envoy_input
}
