#
# Example RBAC DB.
# Replace with yours or import is as an external data bundle---e.g.
# by making OPA download a tarball from a Web server or by linking
# it from a local disk.
#

package httbin.rbacdb

import data.authnz.http as http


# Role defs.
# example researchers := "researchers"
role1 := "researchers"

# User defs
# example jeejee := "jeejee@teadal.eu"





# Map each role to a list of permission objects.
# Each permission object specifies a set of allowed HTTP methods for
# the Web resources identified by the URLs matching the given regex.
role_based_permissions := {
	role1: [
 	 { 
 	 	 "methods": http.read, 
 	 	 "url_regex": "^/anything/.*"
 	 },
 	],

}

user_based_permissions := {

}

## permissions Example
## researchers|jeejee: [
##        {
##            "methods": http.do_anything,
##            "url_regex": "^/httpbin/anything/.*"
##        },
##    ]
