# REGO generator

This application analyse an OpenAPI document of the SFDP, thus it reflects the agreement between the data provider and data consumer. This OpenAPI is obtained from the FDP OpenAPI with some extensions that contains information about the applied policies.

At this stage only RBAC policies are embedded by means of the following teadal specific attributes:
- x-teadal-users-allowed
- x-teadal-roles-allowed
- x-teadal-IAM-provider

To run the application

```
go run main.go <service_name> <openAPI_file> [policy_dir]

<service_name> is the name of the FDP
<openAPI_file> file containing the extended openAPI

[policy_dir] (Optional) the name of subdirectory of 'output' dir where to store the generated rego file
```
