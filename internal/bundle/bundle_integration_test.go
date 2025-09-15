package bundle

import (
	"context"
	policy "dspn-regogenerator/internal/policy"
	_ "embed"
	"testing"

	"github.com/open-policy-agent/opa/v1/ast"
	opabundle "github.com/open-policy-agent/opa/v1/bundle"
	"github.com/open-policy-agent/opa/v1/storage"
	"github.com/open-policy-agent/opa/v1/storage/inmem"
	"github.com/open-policy-agent/opa/v1/tester"
)

//go:embed testdata/bundle_test.rego
var testFile string

func TestBundleExecution(t *testing.T) {
	service := Service{
		name:    "httpbin",
		oidcUrl: "https://accounts.google.com/.well-known/openid-configuration",
		policy:  policy.GeneralPolicies{},
	}

	b, err := New(service)
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
