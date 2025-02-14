#
# Base `authnz` config.
# See `authnz.config` for explanations.
#
package config.{{OIDC_NAME}}


internal_keycloak_jwks_url := "http://keycloak:8080/keycloak/realms/teadal/protocol/openid-connect/certs"

jwks_preferred_urls := {
    "http://{{DNS_OR_IP}}": internal_keycloak_jwks_url,
    "https://{{DNS_OR_IP}}": internal_keycloak_jwks_url
}

jwt_user_field_name := "email"
