#
# Functions to query and evaluate RBAC REST policies.
#
# All the functions below take a RBAC DB and a user as input. The RBAC DB
# must be in the format specified in RBAC DB test. The user is a username
# string which may or may not be one of the usernames defined in the RBAC
# DB.
#

package authnz.rbac

import future.keywords.in

# Find all the roles associated to the given user.
#user_roles(rbac_db, user) := roles {
#	roles := rbac_db.user_to_roles[user]
#}

# Find all the permissions associated the the given role.
role_perms(rbac_db, role) := perms {
	#print(role)
	#print(rbac_db.role_based_permissions)
	perms := rbac_db.role_based_permissions[role]
}

# Find all the permissions associated the the given user.
user_perms(rbac_db, user) := perms {
	#print("rbacdb")
	#print(rbac_db)
	#print("user ", user)
	perms := rbac_db.user_based_permissions[user]
	#print(perms)
}

# Find all the permissions associated the the given user.
#user_perms(rbac_db, user) := perms {
#roles := user_roles(rbac_db, user)
#perm_sets := {rbac_db.role_to_perms[k] | roles[k]}
#perms := union(perm_sets)
#}

# Check the given user is allowed to carry out the requested operation
# (HTTP method) on the target resource.
#
# The request param must be an object containing `method` and `path`
# fields. `method` is the HTTP request method whereas `path` is the
# HTTP request path. Typically, when using the OPA Envoy plugin, you'd
# pass in `input.attributes.request.http` for the request param.
check_user_permissions(rbac_db, user, request) {
	# check if the user has some rights

	#perm := rbac_db.user_based_permissions[user][_]	
	#perm.methods[_] == request.method
	#print(perm.url_regex, request.path, regex.match(perm.url_regex, request.path))
	#regex.match(perm.url_regex, request.path)

	print(rbac_db.user_based_permissions[user])
	some perm in rbac_db.user_based_permissions[user]
	print(perm)
	request.method in perm.methods
	print(perm.url_regex, request.path, regex.match(perm.url_regex, request.path))
	regex.match(perm.url_regex, request.path)
}

check_roles_permissions(rbac_db, roles1, request) {
	some rolez in roles1
	check_role_permissions(rbac_db.role_based_permissions[rolez], request)
}

check_role_permissions(perms, request) {
	some perm in perms
	request.method in perm.methods
	print(perm.url_regex, request.path, regex.match(perm.url_regex, request.path))
	regex.match(perm.url_regex, request.path)
}
