#
# Policy for the httpbin.
#

package httpbin.service

import input.attributes.request.http as http_request
import data.authnz.envopa as envopa
import data.httpbin.oidc as oidc_config
import data.httpbin.rbacdb as rbac_db


default allow := false

allow = true if {
    envopa.allow_user(rbac_db, oidc_config)

}

#OR 

allow = true if {

    envopa.allow_role(rbac_db, oidc_config)

}
