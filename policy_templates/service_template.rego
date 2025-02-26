#
# Policy for the {{SERVICE_NAME}}.
#

package {{SERVICE_NAME}}.service

import input.attributes.request.http as http_request
import data.authnz.envopa as envopa
import data.{{SERVICE_NAME}}.oidc as oidc_config
import data.{{SERVICE_NAME}}.rbacdb as rbac_db


default allow := false

allow = true {
    envopa.allow_user(rbac_db, oidc_config)

}

#OR 

allow = true {

    envopa.allow_role(rbac_db, oidc_config)

}
