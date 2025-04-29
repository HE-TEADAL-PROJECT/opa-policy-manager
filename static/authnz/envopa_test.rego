package authnz.envopa

import rego.v1

import data.authnz.envopa as envopa
import data.authnz.http as http
import data.authnz.oidc as oidc
import data.authnz.rbac as rbac
import data.authnz.rbacdb as rbac_db
import data.httpbin.oidc as oidc_config

http_request_example := {
	"headers": {
		":authority": "localhost",
		":method": "GET",
		":path": "/httpbin/brotli",
		":scheme": "http",
		"accept": "*/*",
		"authorization": "Bearer",
		"user-agent": "curl/8.1.1",
		"x-envoy-decorator-operation": "httpbin.default.svc.cluster.local:8000/httpbin/*",
		"x-envoy-internal": "true",
		"x-envoy-peer-metadata": "ChQKDkFQUF9DT05UQUlORVJTEgIaAAoaCgpDTFVTVEVSX0lEEgwaCkt1YmVybmV0ZXMKGwoMSU5TVEFOQ0VfSVBTEgsaCTEwLjEuNDIuNgoZCg1JU1RJT19WRVJTSU9OEggaBjEuMTguMAqcAwoGTEFCRUxTEpEDKo4DCh0KA2FwcBIWGhRpc3Rpby1pbmdyZXNzZ2F0ZXdheQoTCgVjaGFydBIKGghnYXRld2F5cwoUCghoZXJpdGFnZRIIGgZUaWxsZXIKNgopaW5zdGFsbC5vcGVyYXRvci5pc3Rpby5pby9vd25pbmctcmVzb3VyY2USCRoHdW5rbm93bgoZCgVpc3RpbxIQGg5pbmdyZXNzZ2F0ZXdheQoZCgxpc3Rpby5pby9yZXYSCRoHZGVmYXVsdAowChtvcGVyYXRvci5pc3Rpby5pby9jb21wb25lbnQSERoPSW5ncmVzc0dhdGV3YXlzChIKB3JlbGVhc2USBxoFaXN0aW8KOQofc2VydmljZS5pc3Rpby5pby9jYW5vbmljYWwtbmFtZRIWGhRpc3Rpby1pbmdyZXNzZ2F0ZXdheQovCiNzZXJ2aWNlLmlzdGlvLmlvL2Nhbm9uaWNhbC1yZXZpc2lvbhIIGgZsYXRlc3QKIgoXc2lkZWNhci5pc3Rpby5pby9pbmplY3QSBxoFZmFsc2UKGgoHTUVTSF9JRBIPGg1jbHVzdGVyLmxvY2FsCi8KBE5BTUUSJxolaXN0aW8taW5ncmVzc2dhdGV3YXktNjhjY2Y4OGM4Ni00Z2d2ZgobCglOQU1FU1BBQ0USDhoMaXN0aW8tc3lzdGVtCl0KBU9XTkVSElQaUmt1YmVybmV0ZXM6Ly9hcGlzL2FwcHMvdjEvbmFtZXNwYWNlcy9pc3Rpby1zeXN0ZW0vZGVwbG95bWVudHMvaXN0aW8taW5ncmVzc2dhdGV3YXkKFwoRUExBVEZPUk1fTUVUQURBVEESAioACicKDVdPUktMT0FEX05BTUUSFhoUaXN0aW8taW5ncmVzc2dhdGV3YXk=",
		"x-envoy-peer-metadata-id": "router~10.1.42.6~istio-ingressgateway-68ccf88c86-4ggvf.istio-system~istio-system.svc.cluster.local",
		"x-forwarded-for": "10.1.42.1",
		"x-forwarded-proto": "http",
		"x-request-id": "be81d7f6-3cd1-976f-afef-ec075ea5667d",
	},
	"host": "localhost",
	"id": "12740781984165034944",
	"method": "GET",
	"path": "/httpbin/brotli",
	"protocol": "HTTP/1.1",
	"scheme": "http",
}

test_allow_user if {
	print(oidc_config.internal_keycloak_jwks_url)

	print(http_request_example)
	print(oidc_config)
	payload := oidc.claims(http_request_example, oidc_config)
	print(payload)
	user := payload[oidc_config.jwt_user_field_name]
	print("checking user permissions for user ")
	print(user)
	print(payload)

	#roles := payload[oidc_config.jwt_realm_access_field_name.jwt_roles_field_name]
	rbac.check_user_permissions(rbac_db, user, http_request_example)
	#rbac.check_roles_permissions(rbac_db, roles, http_request_example)

}
