#
# TODO docs
#

package authnz.envopa

import data.authnz.oidc as oidc
import data.authnz.rbac as rbac
import input.attributes.request.http as http_request

allow_user(rbac_db, config) {
	payload := oidc.claims(http_request, config)
	user := payload[config.jwt_user_field_name]
	#roles := payload[config.jwt_realm_access_field_name.jwt_roles_field_name]
	rbac.check_user_permissions(rbac_db, user, http_request)
	#rbac.check_roles_permissions(rbac_db, roles, http_request)
}

allow_role(rbac_db, config) {
	payload := oidc.claims(http_request, config)
	roles := payload[config.jwt_realm_access_field_name.jwt_roles_field_name]
	role := rbac.check_roles_permissions(rbac_db, roles, http_request)
}
