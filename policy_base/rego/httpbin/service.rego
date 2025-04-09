#
# Policy for the httpbin.
#

package httpbin.service

import rego.v1

import data.authnz.envopa as envopa
import data.httpbin.oidc as oidc_config
import data.httpbin.rbacdb as rbac_db
import input.attributes.request.http as http_request

default allow := false

allow if {
	envopa.allow_user(rbac_db, oidc_config)
}

#OR

allow if {
	envopa.allow_role(rbac_db, oidc_config)
}
