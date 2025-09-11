package envoy.authz

import input.attributes.request.http

default allow = false

allow if {
    print("HTTP Method: ", http.method)
    http.method == "GET"
}
