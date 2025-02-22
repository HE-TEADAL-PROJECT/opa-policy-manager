#
# Base `authnz` config.
# See `authnz.config` for explanations.
#
package config.oidc_petstore


internal_keycloak_jwks_url := "http://keycloak:8080/keycloak/realms/teadal/protocol/openid-connect/certs"

jwks_preferred_urls := {
    "http://baseline.teadal.ubiwhere.eu": internal_keycloak_jwks_url,
    "https://baseline.teadal.ubiwhere.eu": internal_keycloak_jwks_url
}

jwt_user_field_name := "email"
