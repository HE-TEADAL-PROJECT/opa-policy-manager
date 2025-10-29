package parser

import (
	"dspn-regogenerator/internal/policy"
	"fmt"
	"os"

	"github.com/pb33f/libopenapi"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
)

const (
	XTeadalPoliciesKey = "x-teadal-policies"
	XTeadalIAMKey      = "x-teadal-IAM-provider"
)

type XTeadalPolicies struct {
	Policies    []policy.PolicyClause `json:"access-policies" yaml:"access-policies"`
	Description string                `json:"description"`
}

type StructuredPolicies = policy.GeneralPolicies

func ParseOpenAPIPolicies(specByteArray []byte) (*StructuredPolicies, error) {
	docModel, err := getDocumentFromData(specByteArray)
	if err != nil {
		return nil, fmt.Errorf("failed to parse OpenAPI spec: %v", err)
	}

	result := policy.NewGeneralPolicies()

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

func ParseOpenAPIIAM(specByteArray []byte) (*string, error) {
	docModel, err := getDocumentFromData(specByteArray)
	if err != nil {
		return nil, fmt.Errorf("failed to parse OpenAPI spec: %v", err)
	}
	// Check if the document has any security requirements
	if docModel.Model.Components.SecuritySchemes.Len() == 0 {
		return nil, fmt.Errorf("no security requirements found in OpenAPI spec")
	}
	if docModel.Model.Components.SecuritySchemes.Len() > 1 {
		fmt.Fprintf(os.Stderr, "Warning: multiple security requirements found in OpenAPI spec\n")
	}
	securityTag := docModel.Model.Components.SecuritySchemes.First()
	for securityTag != nil && securityTag.Key() != "bearerAuth" {
		securityTag = securityTag.Next()
	}
	if securityTag == nil {
		return nil, fmt.Errorf("bearerAuth security requirement not found in OpenAPI spec")
	}
	// Decode the security requirement value
	exts := securityTag.Value().Extensions.First()
	for exts != nil && exts.Key() != XTeadalIAMKey {
		exts = exts.Next()
	}
	if exts == nil {
		return nil, fmt.Errorf(XTeadalIAMKey + "extension not found in bearerAuth security requirement")
	}
	url := ""
	err = exts.Value().Decode(&url)
	if err != nil {
		return nil, fmt.Errorf("failed to decode value for bearerAuth security requirement: %v", err)
	}
	return &url, nil
}

func getDocumentFromData(specByteArray []byte) (*libopenapi.DocumentModel[v3.Document], error) {
	document, err := libopenapi.NewDocument(specByteArray)
	if err != nil {
		return nil, fmt.Errorf("failed to parse OpenAPI spec: %v", err)
	}
	docModel, errors := document.BuildV3Model()
	if len(errors) > 0 {
		for i := range errors {
			fmt.Printf("error: %e\n", errors[i])
		}
		panic(fmt.Sprintf("cannot create v3 model from document: %d errors reported", len(errors)))
	}
	return docModel, nil
}
