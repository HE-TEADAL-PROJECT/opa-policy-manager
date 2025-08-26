package parser

import (
	"dspn-regogenerator/internal/policy"
	"fmt"
	"io"
	"os"

	"github.com/pb33f/libopenapi"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
)

type ServiceSpec struct {
	Policies         StructuredPolicies
	IdentityProvider string
}

const (
	PolicyExtensionTag = "x-teadal-policies"
	IamExtensionTag    = "x-teadal-IAM-provider"
)

type XTeadalPolicies struct {
	Policies    []policy.PolicyClause `json:"policies"`
	Description string                `json:"description"`
}

type StructuredPolicies = policy.GeneralPolicies

type BuildingModelError struct {
	errors []error
}

func (e *BuildingModelError) Error() string {
	if len(e.errors) == 0 {
		return "no errors"
	}
	result := "errors occurred while building model:\n"
	for _, err := range e.errors {
		result += fmt.Sprintf("- %v\n", err)
	}
	return result
}

func parseOpenAPIDocument(spec io.Reader) (*libopenapi.DocumentModel[v3.Document], error) {
	specByteArray, err := io.ReadAll(spec)
	if err != nil {
		return &libopenapi.DocumentModel[v3.Document]{}, fmt.Errorf("failed to read OpenAPI spec: %v", err)
	}

	document, err := libopenapi.NewDocument(specByteArray)
	if err != nil {
		return &libopenapi.DocumentModel[v3.Document]{}, fmt.Errorf("failed to parse OpenAPI spec: %v", err)
	}

	docModel, errors := document.BuildV3Model()
	if len(errors) > 0 {
		errorList := make([]error, len(errors))
		for i, err := range errors {
			errorList[i] = fmt.Errorf("error %d: %w", i+1, err)
		}
		return &libopenapi.DocumentModel[v3.Document]{}, &BuildingModelError{errors: errorList}
	}

	return docModel, nil
}

func getPolicies(docModel *libopenapi.DocumentModel[v3.Document]) (*StructuredPolicies, error) {
	result := policy.NewGeneralPolicies()
	var err error

	// Processing general policies
	if docModel.Model.Paths.Extensions.Len() > 1 {
		fmt.Fprintf(os.Stderr, "Warning: multiple extensions found in paths\n")
	}
	pathsTag := docModel.Model.Paths.Extensions.First()
	for pathsTag != nil && pathsTag.Key() != "x-teadal-policies" {
		pathsTag = pathsTag.Next()
	}
	if pathsTag != nil {
		decodedTag := new(XTeadalPolicies)
		err = pathsTag.Value().Decode(decodedTag)
		if err != nil {
			return nil, fmt.Errorf("failed to decode value for path %s: %v", pathsTag.Key(), err)
		}
		result.Policies = decodedTag.Policies
	}

	// Processing specialized policies
	for path := docModel.Model.Paths.PathItems.First(); path != nil; path = path.Next() {
		// Check if the path has more than one extension
		if path.Value().Extensions.Len() > 1 {
			fmt.Fprintf(os.Stderr, "Warning: multiple extensions found in path %s\n", path.Key())
		}
		// Decode the extension value if it exists
		pathTag := path.Value().Extensions.First()
		for pathTag != nil && pathTag.Key() != "x-teadal-policies" {
			pathTag = pathTag.Next()
		}
		if pathTag != nil {
			decodedTag := new(XTeadalPolicies)
			err = pathTag.Value().Decode(decodedTag)
			if err != nil {
				return nil, fmt.Errorf("failed to decode value for path %s: %v", path.Key(), err)
			}
			result.SpecializedPaths[path.Key()] = policy.PathPolicies{
				Policies: decodedTag.Policies,
				Path:     path.Key(),
			}
		}

		// Check if the path has any methods with extensions
		for method := path.Value().GetOperations().First(); method != nil; method = method.Next() {
			// Check if the method has more than one extension
			if method.Value().Extensions.Len() > 1 {
				fmt.Fprintf(os.Stderr, "Warning: multiple extensions found in method %s in path %s\n", method.Key(), path.Key())
			}
			// Decode the extension value if it exists
			methodTag := method.Value().Extensions.First()
			for methodTag != nil && methodTag.Key() != "x-teadal-policies" {
				methodTag = methodTag.Next()
			}
			if methodTag != nil {
				decodedTag := new(XTeadalPolicies)
				err = methodTag.Value().Decode(decodedTag)
				if err != nil {
					return nil, fmt.Errorf("failed to decode value for method %s in path %s: %v", method.Key(), path.Key(), err)
				}

				// Update the specialized path policies
				var pathPolicies policy.PathPolicies
				if _, ok := result.SpecializedPaths[path.Key()]; !ok {
					pathPolicies = policy.PathPolicies{
						Policies:           []policy.PolicyClause{},
						Path:               path.Key(),
						SpecializedMethods: make(map[string]policy.PathMethodPolicies),
					}
				} else {
					pathPolicies = result.SpecializedPaths[path.Key()]
				}
				if pathPolicies.SpecializedMethods == nil {
					pathPolicies.SpecializedMethods = make(map[string]policy.PathMethodPolicies)
				}
				pathPolicies.SpecializedMethods[method.Key()] = policy.PathMethodPolicies{
					Policies: decodedTag.Policies,
					Method:   method.Key(),
				}
				result.SpecializedPaths[path.Key()] = pathPolicies
			}
		}
	}

	return result, nil
}

func getIdentityProviderTag(docModel *libopenapi.DocumentModel[v3.Document]) (string, error) {
	// Check if the document has any security requirements
	if docModel.Model.Components.SecuritySchemes.Len() == 0 {
		return "", fmt.Errorf("no security requirements found in OpenAPI spec")
	}
	var err error
	if docModel.Model.Components.SecuritySchemes.Len() > 1 {
		err = fmt.Errorf("multiple security requirements found in OpenAPI spec, expected only one")
	}
	securityTag := docModel.Model.Components.SecuritySchemes.First()
	for securityTag != nil && securityTag.Key() != "bearerAuth" {
		securityTag = securityTag.Next()
	}
	if securityTag == nil {
		return "", fmt.Errorf("bearerAuth security requirement not found in OpenAPI spec")
	}
	// Decode the security requirement value
	exts := securityTag.Value().Extensions.First()
	for exts != nil && exts.Key() != IamExtensionTag {
		exts = exts.Next()
	}
	if exts == nil {
		return "", fmt.Errorf(IamExtensionTag + "extension not found in bearerAuth security requirement")
	}
	url := ""
	err = exts.Value().Decode(&url)
	if err != nil {
		return "", fmt.Errorf("failed to decode value for bearerAuth security requirement: %v", err)
	}
	return url, err
}

// ParseServiceSpec reads an OpenAPI specification from the provided [io.Reader],
// extracts the policies and identity provider information, and returns a [ServiceSpec].
func ParseServiceSpec(spec io.Reader) (*ServiceSpec, error) {
	docModel, err := parseOpenAPIDocument(spec)
	if err != nil {
		return nil, err
	}

	policies, err := getPolicies(docModel)
	if err != nil {
		return nil, err
	}

	idProvider, err := getIdentityProviderTag(docModel)
	if err != nil {
		return nil, err
	}

	return &ServiceSpec{
		Policies:         *policies,
		IdentityProvider: idProvider,
	}, nil
}
