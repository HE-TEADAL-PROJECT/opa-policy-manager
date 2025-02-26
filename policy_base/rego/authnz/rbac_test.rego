package authnz.rbac

import data.authnz.http as http
import data.authnz.rbacdb as rbac_db

#test_role_lookup {
#	user_roles(rbac_db, "jeejee") == {"product_owner", "product_consumer"}
#	user_roles(rbac_db, "sebs") == {"product_consumer"}
#}

test_role_perms {
	role_perms(rbac_db, "researchers") == [
		{
			"methods": http.read,
			"url_regex": "^/anything/.*",
		},
		{
			"methods": http.read,
			"url_regex": "^/basic-auth/{user}/{passwd}/.*",
		},
	]
	role_perms(rbac_db, "doctors") == [
		{
			"methods": http.read,
			"url_regex": "^/basic-auth/{user}/{passwd}/.*",
		},
		{
			"methods": http.read,
			"url_regex": "^/base64/{value}/.*",
		},
	]
}

test_user_perms {
	user_perms(rbac_db, "jeejee@teadal.eu") == [{
		"methods": http.read,
		"url_regex": "^/bearer/.*",
	}]
	user_perms(rbac_db, "sebs@teadal.eu") == [{
		"methods": http.read,
		"url_regex": "^/brotli/.*",
	}]
}

assert_user_can_do_anything_on_path(user, path) {
	check_user_permissions(rbac_db, user, {"method": "GET", "path": path})
	check_user_permissions(rbac_db, user, {"method": "HEAD", "path": path})
	check_user_permissions(rbac_db, user, {"method": "OPTIONS", "path": path})
	check_user_permissions(rbac_db, user, {"method": "PUT", "path": path})
	check_user_permissions(rbac_db, user, {"method": "POST", "path": path})
	check_user_permissions(rbac_db, user, {"method": "PATCH", "path": path})
	check_user_permissions(rbac_db, user, {"method": "DELETE", "path": path})
	check_user_permissions(rbac_db, user, {"method": "CONNECT", "path": path})
	check_user_permissions(rbac_db, user, {"method": "TRACE", "path": path})
}

assert_role_can_do_anything_on_path(roles, path) {
	check_roles_permissions(rbac_db, roles, {"method": "GET", "path": path})
	check_roles_permissions(rbac_db, roles, {"method": "HEAD", "path": path})
	check_roles_permissions(rbac_db, roles, {"method": "OPTIONS", "path": path})
	check_roles_permissions(rbac_db, roles, {"method": "PUT", "path": path})
	check_roles_permissions(rbac_db, roles, {"method": "POST", "path": path})
	check_roles_permissions(rbac_db, roles, {"method": "PATCH", "path": path})
	check_roles_permissions(rbac_db, roles, {"method": "DELETE", "path": path})
	check_roles_permissions(rbac_db, roles, {"method": "CONNECT", "path": path})
	check_roles_permissions(rbac_db, roles, {"method": "TRACE", "path": path})
}

assert_user_can_only_read_path(user, path) {
	check_user_permissions(rbac_db, user, {"method": "GET", "path": path})
	check_user_permissions(rbac_db, user, {"method": "HEAD", "path": path})
	check_user_permissions(rbac_db, user, {"method": "OPTIONS", "path": path})
	not check_user_permissions(rbac_db, user, {"method": "PUT", "path": path})
	not check_user_permissions(rbac_db, user, {"method": "POST", "path": path})
	not check_user_permissions(rbac_db, user, {"method": "PATCH", "path": path})
	not check_user_permissions(rbac_db, user, {"method": "DELETE", "path": path})
	not check_user_permissions(rbac_db, user, {"method": "CONNECT", "path": path})
	not check_user_permissions(rbac_db, user, {"method": "TRACE", "path": path})
}

assert_role_can_only_read_path(roles, path) {
	check_roles_permissions(rbac_db, roles, {"method": "GET", "path": path})
	check_roles_permissions(rbac_db, roles, {"method": "HEAD", "path": path})
	check_roles_permissions(rbac_db, roles, {"method": "OPTIONS", "path": path})
	not check_roles_permissions(rbac_db, roles, {"method": "PUT", "path": path})
	not check_roles_permissions(rbac_db, roles, {"method": "POST", "path": path})
	not check_roles_permissions(rbac_db, roles, {"method": "PATCH", "path": path})
	not check_roles_permissions(rbac_db, roles, {"method": "DELETE", "path": path})
	not check_roles_permissions(rbac_db, roles, {"method": "CONNECT", "path": path})
	not check_roles_permissions(rbac_db, roles, {"method": "TRACE", "path": path})
}

test_check_perms {
	#assert_user_can_do_anything_on_path("jeejee@teadal.eu", "/httpbin/anything/")
	#assert_user_can_only_read_path("sebs@teadal.eu", "/httpbin/get")
	#assert_role_can_do_anything_on_path(["doctors"], "/httpbin/anything/")
	#assert_role_can_do_anything_on_path(["researchers"], "/anything/")
	assert_role_can_only_read_path(["researchers"], "/anything/")
}
