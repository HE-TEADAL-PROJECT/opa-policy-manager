package bundle

import (
	"bytes"
	"context"
	policy "dspn-regogenerator/internal/policy"
	_ "embed"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"testing"

	"github.com/open-policy-agent/opa/v1/ast"
	opabundle "github.com/open-policy-agent/opa/v1/bundle"
	"github.com/open-policy-agent/opa/v1/storage"
	"github.com/open-policy-agent/opa/v1/storage/inmem"
	"github.com/open-policy-agent/opa/v1/tester"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/log"
	"github.com/testcontainers/testcontainers-go/wait"

	keycloak "github.com/stillya/testcontainers-keycloak"
)

//go:embed testdata/bundle_test.rego
var testFile string

func TestBundleExecution(t *testing.T) {
	service := Service{
		name:    "httpbin",
		oidcUrl: "https://accounts.google.com/.well-known/openid-configuration",
		policy:  policy.GeneralPolicies{},
	}

	b, err := New(&service)
	if err != nil {
		t.Fatalf("NewBundleFromService() error = %v", err)
	}

	testModule := ast.MustParseModule(testFile)
	b.bundle.Modules = append(b.bundle.Modules, opabundle.ModuleFile{
		URL:    "/httpbin/test.rego",
		Raw:    []byte(testFile),
		Parsed: testModule,
	})

	testRunner := tester.NewRunner()
	testRunner.SetBundles(map[string]*opabundle.Bundle{
		"test": b.bundle,
	})
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() {
		cancel()
	})
	store := inmem.New()
	testRunner.SetStore(store)
	txn := storage.NewTransactionOrDie(ctx, store, storage.WriteParams)
	defer store.Abort(ctx, txn)

	result, err := testRunner.RunTests(ctx, txn)
	if err != nil {
		t.Fatalf("RunTests() error: %v", err)
	}
	var atLeastOne bool
	for r := range result {
		atLeastOne = true
		if r.Fail {
			t.Errorf("Test %s failed: row %v", r.Name, r.Location.Row)
		} else {
			t.Logf("Test %s passed", r.Name)
		}
	}
	if !atLeastOne {
		t.Error("No tests were executed")
	}
}

//go:embed testdata/oidc_test.rego
var oidcTest string

// setupKeycloakContainer starts a Keycloak container and returns the auth server URL
func setupKeycloakContainer(t *testing.T, ctx context.Context) string {
	t.Helper()

	t.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true")
	container, err := keycloak.Run(ctx, "quay.io/keycloak/keycloak:latest",
		keycloak.WithAdminUsername("admin"),
		keycloak.WithAdminPassword("admin"),
		keycloak.WithContextPath("/keycloak"),
		keycloak.WithRealmImportFile("../../config/keycloak/teadal-bootstrap.json"),
		testcontainers.WithWaitStrategy(wait.ForListeningPort("8080/tcp")),
		testcontainers.WithLogger(log.TestLogger(t)),
	)
	if err != nil {
		t.Fatalf("Failed to start Keycloak container: %v", err)
	}

	t.Cleanup(func() {
		if err := container.Terminate(ctx); err != nil {
			t.Fatalf("Failed to terminate container: %s", err)
		}
	})

	endpoint, err := container.GetAuthServerURL(ctx)
	if err != nil {
		t.Fatalf("Failed to get Keycloak endpoint: %v", err)
	}

	return endpoint
}

// verifyOIDCEndpoint checks if the OIDC configuration endpoint is accessible
func verifyOIDCEndpoint(t *testing.T, endpoint string) {
	t.Helper()

	oidcConfigURL := endpoint + "/realms/teadal/.well-known/openid-configuration"
	response, err := http.Get(oidcConfigURL)
	if err != nil {
		t.Fatalf("Failed to reach Keycloak OIDC endpoint: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("Unexpected status code from Keycloak OIDC endpoint: %d", response.StatusCode)
	}
}

// getAccessToken retrieves an access token from Keycloak using password grant
func getAccessToken(t *testing.T, endpoint string) string {
	t.Helper()

	form := url.Values{}
	form.Add("client_id", "admin-cli")
	form.Add("username", "jeejee@teadal.eu")
	form.Add("password", "abc123")
	form.Add("grant_type", "password")

	tokenURL := endpoint + "/realms/teadal/protocol/openid-connect/token"
	tokenResponse, err := http.PostForm(tokenURL, form)
	if err != nil {
		t.Fatalf("Failed to get token from Keycloak: %v", err)
	}
	defer tokenResponse.Body.Close()

	body, err := io.ReadAll(tokenResponse.Body)
	if err != nil {
		t.Fatalf("Failed to read token response body: %v", err)
	}

	if tokenResponse.StatusCode != http.StatusOK {
		t.Fatalf("Failed to get token from Keycloak, status code: %d, body: %s",
			tokenResponse.StatusCode, string(body))
	}

	var tokenData struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(body, &tokenData); err != nil {
		t.Fatalf("Failed to unmarshal token response: %v", err)
	}

	return tokenData.AccessToken
}

// prepareOIDCModule creates the OIDC Rego module using the template
func prepareOIDCModule(t *testing.T, endpoint string) *ast.Module {
	t.Helper()

	var oidcBuffer bytes.Buffer
	templateData := OIDDTemplateData{
		ServiceName: "testservice",
		MetadataURL: endpoint + "/realms/teadal/.well-known/openid-configuration",
	}

	if err := oidcTemplate.Execute(&oidcBuffer, templateData); err != nil {
		t.Fatalf("Failed to execute OIDC template: %v", err)
	}

	return ast.MustParseModule(oidcBuffer.String())
}

// prepareTestModule creates the test module with the access token
func prepareTestModule(t *testing.T, accessToken string) *ast.Module {
	t.Helper()

	const tokenInputVariable = "encoded_token"
	testContent := oidcTest + "\n" + tokenInputVariable + ` := "` + accessToken + "\"\n"
	return ast.MustParseModule(testContent)
}

// runTests executes the OIDC tests and validates results
func runTests(t *testing.T, ctx context.Context, modules map[string]*ast.Module) {
	t.Helper()

	testRunner := tester.NewRunner()
	testRunner.SetModules(modules)

	store := inmem.New()
	testRunner.SetStore(store)
	txn := storage.NewTransactionOrDie(ctx, store, storage.WriteParams)
	defer store.Abort(ctx, txn)

	result, err := testRunner.RunTests(ctx, txn)
	if err != nil {
		t.Fatalf("RunTests() error: %v", err)
	}

	var testsExecuted bool
	for r := range result {
		testsExecuted = true
		if r.Fail {
			t.Errorf("Test %s failed at line %v", r.Name, r.Location.Row)
		} else {
			t.Logf("Test %s passed", r.Name)
		}
		if len(r.Output) > 0 {
			t.Logf("Test %s output: %s", r.Name, string(r.Output))
		}
	}

	if !testsExecuted {
		t.Error("No tests were executed")
	}
}

func TestOidcIntegration(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	// Setup Keycloak container
	endpoint := setupKeycloakContainer(t, ctx)

	// Verify OIDC endpoint is accessible
	verifyOIDCEndpoint(t, endpoint)

	// Get access token
	accessToken := getAccessToken(t, endpoint)

	// Prepare OIDC and test modules
	oidcModule := prepareOIDCModule(t, endpoint)
	testModule := prepareTestModule(t, accessToken)

	// Run tests
	runTests(t, ctx, map[string]*ast.Module{
		"oidc.rego": oidcModule,
		"test.rego": testModule,
	})
}
