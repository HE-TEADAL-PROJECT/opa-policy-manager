#
# Policy for the petstore.
#

package petstore.service

import input.attributes.request.http as http_request
import data.authnz.envopa as envopa
import data.config.oidc_petstore as oidc_config
import data.petstore.rbacdb as rbac_db


default allow := false

allow = true {
    envopa.allow_user(rbac_db, oidc_config)

    # Put below this line any service-specific checks on e.g. http_request

}
