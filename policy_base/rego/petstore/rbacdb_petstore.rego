#
# Example RBAC DB.
# Replace with yours or import is as an external data bundle---e.g.
# by making OPA download a tarball from a Web server or by linking
# it from a local disk.
#

package petstore.rbacdb

import data.authnz.http as http


# Role defs.
# example researchers := "researchers"
role1 := "researchers"

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
 	 	 "url_regex": "^/pets/.*"
 	 },
 	],

}

user_based_permissions := {
	user1: [
 	 { 
 	 	 "methods": http.read, 
 	 	 "url_regex": "^/pets/{petId}/.*"
 	 },
 	 { 
 	 	 "methods": http.write, 
 	 	 "url_regex": "^/pets/.*"
 	 },
 	],
	user2: [
 	 { 
 	 	 "methods": http.write, 
 	 	 "url_regex": "^/pets/.*"
 	 },
 	],

}

## permissions Example
## researchers|jeejee: [
##        {
##            "methods": http.do_anything,
##            "url_regex": "^/httpbin/anything/.*"
##        },
##    ]
