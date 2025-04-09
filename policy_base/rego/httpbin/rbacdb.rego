#
# Example RBAC DB.
# Replace with yours or import is as an external data bundle---e.g.
# by making OPA download a tarball from a Web server or by linking
# it from a local disk.
#

package httpbin.rbacdb

import rego.v1

import data.authnz.http as http

# Role defs.
# example researchers := "researchers"
role1 := "doctors"

role2 := "researchers"

# User defs
# example jeejee := "jeejee@teadal.eu"
user1 := "jeejee@teadal.eu"

user2 := "sebs@teadal.eu"

# Map each role to a list of permission objects.
# Each permission object specifies a set of allowed HTTP methods for
# the Web resources identified by the URLs matching the given regex.
role_based_permissions := {
	role1: [
		{
			"methods": http.read,
			"url_regex": "^/httpbin/basic-auth/{user}/{passwd}/.*",
		},
		{
			"methods": http.read,
			"url_regex": "^/httpbin/base64/{value}/.*",
		},
	],
	role2: [
		{
			"methods": http.read,
			"url_regex": "^/httpbin/basic-auth/{user}/{passwd}/.*",
		},
		{
			"methods": http.read,
			"url_regex": "^/httpbin/anything/.*",
		},
	],
}

user_based_permissions := {
	user1: [{
		"methods": http.read,
		"url_regex": "^/httpbin/bearer/.*",
	}],
	user2: [{
		"methods": http.read,
		"url_regex": "^/httpbin/brotli/.*",
	}],
}

## permissions Example
## researchers|jeejee: [
##        {
##            "methods": http.do_anything,
##            "url_regex": "^/httpbin/anything/.*"
##        },
##    ]
