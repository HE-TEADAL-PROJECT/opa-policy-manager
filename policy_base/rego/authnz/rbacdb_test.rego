#
# Test RBAC DB.
#

package authnz.rbacdb

import data.authnz.http as http

# Role defs.
# example researchers := "researchers"
role1 := "researchers"
role2 := "doctors"

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
			"methods": http.do_anything,
			"url_regex": "^/httpbin/anything",
		},
	 {
			"methods": http.do_anything,
			"url_regex": "^/httpbin/get$",
	 },
 	],
	role2: [
 	 {
			"methods": http.read,
			"url_regex": "^/httpbin/get$",
		},
 	]
}

user_based_permissions := {
	user1: [
 	 {
			"methods": http.do_anything,
			"url_regex": "^/httpbin/anything/.*",
		},
		{
			"methods": http.read,
			"url_regex": "^/httpbin/get$",
		},
 	],
	user2: [
 	 {
			"methods": http.read,
			"url_regex": "^/httpbin/get$",
		}
 	],
}

## permissions Example
## researchers|jeejee: [
##        {
##            "methods": http.do_anything,
##            "url_regex": "^/httpbin/anything/.*"
##        },
##    ]
