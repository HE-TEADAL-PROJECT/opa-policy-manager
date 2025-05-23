# OPA Policy Manager (REGO Generator)

This application analyzes an OpenAPI document for a Service, which reflects the agreement between a data provider and data consumer. This OpenAPI specification is an extension of the standard OpenAPI, incorporating information about the policies to be applied.

Currently, the primary focus is on Role-Based Access Control (RBAC) policies, embedded using the following Teadal-specific attributes within the OpenAPI specification:
- `x-teadal-policies`
- `x-teadal-IAM-provider`

The OPA Policy Manager generates REGO policies based on these OpenAPI specifications. These policies can then be used by OPA (Open Policy Agent) to enforce access control.

## How to Use

There are two main ways to interact with the OPA Policy Manager:

1.  **Command Line Interface (CLI)**: For local testing, policy generation, and management.
2.  **Web Service**: For managing policies through an HTTP API, suitable for integration with running instances.

---

## 1. Command Line Interface (CLI)

The CLI allows you to generate and manage OPA policies directly from your terminal.

**Base command:**
```bash
go run ./cmd/cli <command> [arguments]
```

### CLI Commands

#### `add`
Adds a new service and generates its OPA policies based on an OpenAPI specification.

**Usage:**
```bash
go run ./cmd/cli add <service_name> <openAPI_file_path> [policy_dir]
```
-   `<service_name>`: The unique name for the service (e.g., `my-api`).
-   `<openAPI_file_path>`: Path to the OpenAPI specification file (e.g., `./testdata/schemas/fdp-amts-gtfs-static-ext.yaml`).
-   `[policy_dir]` (Optional): The name of the subdirectory within the `output/rego/` directory where the generated REGO files for this service will be stored. If not provided, it defaults to `<service_name>`.

**Example:**
```bash
go run ./cmd/cli add my-service ./testdata/schemas/httpbin-api.json myservicepolicies
```
This command will:
1.  Parse `./testdata/schemas/httpbin-api.json`.
2.  Generate REGO policies for the service `my-service`.
3.  Store the generated policies in `output/rego/myservicepolicies/`.
4.  Update the main REGO policy and the data bundle.

#### `list`
Lists all the services currently managed by the OPA Policy Manager.

**Usage:**
```bash
go run ./cmd/cli list
```

**Example:**
```bash
go run ./cmd/cli list
```
Output will be a list of service names.

#### `delete`
Deletes a service and its associated OPA policies.

**Usage:**
```bash
go run ./cmd/cli delete <service_name>
```
-   `<service_name>`: The name of the service to delete.

**Example:**
```bash
go run ./cmd/cli delete my-service
```
This command will remove the policies for `my-service` and update the main REGO policy and data bundle.

#### `get`
Retrieves the latest bundle and store in provided path (default "./output"). Use for debug and inspection of latest bundle

**Usage (Conceptual):**
```bash
go run ./cmd/cli get [--output <path>]
```
-   `<path>`: Where to store the donwloaded bundle

**Example (Conceptual):**
```bash
go run ./cmd/cli get my-service
```

#### `test`
Tests the generated policies for a service. (This command might need to be implemented or clarified based on its exact functionality, e.g., running `opa test` against the generated files).

**Usage (Conceptual):**
```bash
go run ./cmd/cli test <service_name>
```
-   `<service_name>`: The name of the service whose policies you want to test.

**Example (Conceptual):**
```bash
go run ./cmd/cli test my-service
```

---

## 2. Web Service

The web service provides HTTP endpoints to manage OPA policies, which is useful when the OPA Policy Manager is run as a persistent service (e.g., in a Docker container).

**Start the web service:**
```bash
go run ./cmd/web
```
By default, the service will listen on port `8080`. If you are running it via the `docker-compose.yml` provided in this project, it's exposed on host port `8888` (this means your `curl` commands targeting the Docker instance should use `8888`, while local `go run` would use `8080`).

### Web Service Endpoints

The following examples assume the web service is accessible at `http://localhost:8080` for local `go run` execution, or `http://localhost:8888` if running via the project's `docker-compose.yml`.

**Note:** The examples below will use `http://localhost:8080`. Adjust to `http://localhost:8888` if you are using the Docker Compose setup.

#### List Services
Lists all managed services.

-   **Endpoint:** `GET /api/policies`
-   **Description:** Retrieves a list of all services for which policies have been generated.
-   **Curl Example:**
    ```bash
    curl http://localhost:8080/api/policies
    ```
-   **Expected Response:**
    ```json
    {
      "services": ["service1", "service2"]
    }
    ```

#### Add Service Policies
Adds a new service and generates its OPA policies using an OpenAPI specification file.

-   **Endpoint:** `PUT /api/policies`
-   **Description:** Uploads an OpenAPI specification for a new service. The request must be `multipart/form-data`.
-   **Form Fields:**
    -   `serviceName`: The unique name for the service.
    -   `openAPISpec`: The OpenAPI specification file.
-   **Curl Example:**
    ```bash
    curl -X PUT -F "serviceName=newapi" -F "openAPISpec=@/path/to/your/openapi.json" http://localhost:8080/api/policies
    ```
    Replace `/path/to/your/openapi.json` with the actual path to your OpenAPI file.
-   **Success Response:** `201 Created`

#### Delete Service Policies
Deletes a service and its associated OPA policies.

-   **Endpoint:** `DELETE /api/policies`
-   **Description:** Removes a service and its policies. The service name should be provided as a form value.
-   **Form Fields (or Query Parameters):**
    -   `serviceName`: The name of the service to delete.
-   **Curl Example:**
    ```bash
    # Using -F for form data (as per current Go handler)
    curl -X DELETE -F "serviceName=newapi" http://localhost:8080/api/policies

    # Alternatively, if the API were to accept it as a query parameter:
    # curl -X DELETE "http://localhost:8080/api/policies?serviceName=newapi"
    ```
-   **Success Response:** `204 No Content`

---

## Policy Storage and Bundling

Generated REGO policies are typically stored in the `output/rego/` directory, with subdirectories for each service.
The application also manages a policy bundle (e.g., `teadal-policy-bundle-LATEST.tar.gz`) which is updated whenever policies are added or deleted. This bundle can be used by OPA to load the policies.
The location of this bundle and its interaction with MinIO (if configured) is handled by the application's internal bundle management.
