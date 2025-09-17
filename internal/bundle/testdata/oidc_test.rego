package oidc_test

import data.testservice.oidc

test_valid_metadata_url if {
    oidc.metadata_url
}

test_valid_metadata if {
    oidc.metadata.jwks_uri
    contains(oidc.metadata.jwks_uri, "/realms/teadal/protocol/openid-connect/certs")
}

test_valid_jwks if {
    count(oidc.jwks.keys) > 0
}

test_token_false if {
    not oidc.token.valid
}

test_token_valid if {
    oidc.token.valid with oidc.encoded as encoded_token
    # encoded_token will be appended during test execution
}

test_token_payload if {
    payload := oidc.token.payload with oidc.encoded as encoded_token
    print(payload)
}
