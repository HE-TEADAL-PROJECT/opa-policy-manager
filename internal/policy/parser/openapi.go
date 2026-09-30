package parser

import (
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"strconv"
	"strings"

	"dspn-regogenerator/internal/policy"

	"github.com/pb33f/libopenapi"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/orderedmap"
	"gopkg.in/yaml.v3"
)

const (
	// XTeadalPoliciesKey = "x-teadal-policies"
	XTeadalPoliciesKey = "x-policies"
	XTeadalIAMKey      = "x-teadal-IAM-provider"
)

type XTeadalPolicies struct {
	Policies    []policy.PolicyClause `yaml:"access-policies"`
	Description string                `yaml:"description"`
}

type StructuredPolicies = policy.GeneralPolicies

func lookupPoliciesTag(exts *orderedmap.Map[string, *yaml.Node]) (*yaml.Node, bool) {
	if exts == nil {
		return nil, false
	}
	return exts.Get(XTeadalPoliciesKey)
}

// extensionKeys returns the names of all extensions declared on a node, in document order
func extensionKeys(exts *orderedmap.Map[string, *yaml.Node]) []string {
	if exts == nil {
		return nil
	}
	return slices.Collect(exts.KeysFromOldest())
}

// logUnexpectedExtensions reports the extension names declared on a node when none of them is the expected x-policies tag
func logUnexpectedExtensions(scope string, exts *orderedmap.Map[string, *yaml.Node]) {
	keys := extensionKeys(exts)
	if _, ok := exts.Get(XTeadalPoliciesKey); ok || len(keys) == 0 {
		return
	}
	slog.Debug("No x-policies tag here, other extensions declared",
		"scope", scope, "declaredExtensions", keys, "expectedExtension", XTeadalPoliciesKey)
}

// decodePoliciesTag decodes an x-policies tag node into its typed representation, logging the location of the tag and the reason on failure.
func decodePoliciesTag(node *yaml.Node, scope string) (*XTeadalPolicies, error) {
	decodedTag := new(XTeadalPolicies)
	if err := node.Decode(decodedTag); err != nil {
		slog.Error("Failed to decode x-policies tag",
			"scope", scope, "extension", XTeadalPoliciesKey,
			"line", node.Line, "column", node.Column, "error", err)
		return nil, fmt.Errorf("failed to decode value for %s: %v", XTeadalPoliciesKey, err)
	}
	if len(decodedTag.Policies) == 0 {
		slog.Warn("x-policies tag declares no access-policies, the tag has no effect",
			"scope", scope, "line", node.Line, "description", decodedTag.Description)
		return decodedTag, nil
	}
	return decodedTag, nil
}

// clauseTypes lists the policy kinds present in a decoded clause, so logs show which parts of a clause were actually recognised by the parser.
func clauseTypes(clause *policy.PolicyClause) []string {
	var types []string
	if clause.UserPolicy != nil {
		types = append(types, "users")
	}
	if clause.RolePolicy != nil {
		types = append(types, "roles")
	}
	if clause.LocationPolicy != nil {
		types = append(types, "location")
	}
	if clause.CallPolicy != nil {
		types = append(types, "calls")
	}
	if clause.TimelinessPolicy != nil {
		types = append(types, "timeliness")
	}
	return types
}

// formatExtensionCounts renders a "key=count, key=count" string, so the summary log stays a single readable line.
func formatExtensionCounts(counts map[string]int) string {
	if len(counts) == 0 {
		return "none"
	}
	keys := slices.Sorted(maps.Keys(counts))
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+strconv.Itoa(counts[key]))
	}
	return strings.Join(parts, ", ")
}

func ParseOpenAPIPolicies(specByteArray []byte) (*StructuredPolicies, error) {
	docModel, err := getDocumentFromData(specByteArray)
	if err != nil {
		return nil, fmt.Errorf("failed to parse OpenAPI spec: %v", err)
	}

	result := policy.NewGeneralPolicies()

	// Counters feeding the summary log at the end of this function.
	pathsScanned := 0
	tagsMatched := 0
	clausesMatched := 0

	extensionCounts := make(map[string]int)
	trackExtensions := func(scope string, exts *orderedmap.Map[string, *yaml.Node]) {
		for _, key := range extensionKeys(exts) {
			extensionCounts[key]++
		}
		logUnexpectedExtensions(scope, exts)
	}

	// Processing global (paths-level) policies.
	trackExtensions("paths", docModel.Model.Paths.Extensions)
	if docModel.Model.Paths.Extensions.Len() > 1 {
		slog.Warn("Multiple extensions found in paths, only the first one is used",
			"extensions", extensionKeys(docModel.Model.Paths.Extensions), "used", XTeadalPoliciesKey)
	}
	if pathsTagNode, ok := lookupPoliciesTag(docModel.Model.Paths.Extensions); ok {
		decodedTag, err := decodePoliciesTag(pathsTagNode, "paths")
		if err != nil {
			return nil, err
		}
		tagsMatched++
		clausesMatched += len(decodedTag.Policies)
		for i := range decodedTag.Policies {
			slog.Debug("x-policies clause decoded from paths-level tag",
				"scope", "paths", "index", i,
				"clauses", len(decodedTag.Policies), "clauseTypes", clauseTypes(&decodedTag.Policies[i]))
		}
		slog.Debug("Found x-policies tag at paths level",
			"line", pathsTagNode.Line, "clauses", len(decodedTag.Policies),
			"description", decodedTag.Description)
		result.Policies = decodedTag.Policies
	}

	// Processing resource- and operation-level (specialized) policies.
	for path := docModel.Model.Paths.PathItems.First(); path != nil; path = path.Next() {
		pathsScanned++
		pathKey := path.Key()
		trackExtensions("path "+pathKey, path.Value().Extensions)
		if path.Value().Extensions.Len() > 1 {
			slog.Warn("Multiple extensions found on path, only the first one is used",
				"path", pathKey, "extensions", extensionKeys(path.Value().Extensions), "used", XTeadalPoliciesKey)
		}
		if pathTagNode, ok := lookupPoliciesTag(path.Value().Extensions); ok {
			decodedTag, err := decodePoliciesTag(pathTagNode, "path "+pathKey)
			if err != nil {
				return nil, fmt.Errorf("failed to decode value for path %s: %v", pathKey, err)
			}
			tagsMatched++
			clausesMatched += len(decodedTag.Policies)
			for i := range decodedTag.Policies {
				slog.Debug("x-policies clause decoded from path-level tag",
					"path", pathKey, "index", i,
					"clauses", len(decodedTag.Policies), "clauseTypes", clauseTypes(&decodedTag.Policies[i]))
			}
			slog.Debug("Found x-policies tag on path",
				"path", pathKey, "line", pathTagNode.Line,
				"clauses", len(decodedTag.Policies), "description", decodedTag.Description)
			result.SpecializedPaths[pathKey] = policy.PathPolicies{
				Policies: decodedTag.Policies,
				Path:     pathKey,
			}
		}

		// Check if the path has any methods with extensions
		for method := path.Value().GetOperations().First(); method != nil; method = method.Next() {
			// Check if the method has more than one extension
			trackExtensions("method "+method.Key()+" of path "+pathKey, method.Value().Extensions)
			if method.Value().Extensions.Len() > 1 {
				slog.Warn("Multiple extensions found on method, only the first one is used",
					"path", pathKey, "method", method.Key(),
					"extensions", extensionKeys(method.Value().Extensions), "used", XTeadalPoliciesKey)
			}
			// Decode the extension value if it exists
			methodTagNode, ok := lookupPoliciesTag(method.Value().Extensions)
			if !ok {
				continue
			}
			methodScope := "method " + method.Key() + " of path " + pathKey
			decodedTag, err := decodePoliciesTag(methodTagNode, methodScope)
			if err != nil {
				return nil, fmt.Errorf("failed to decode value for method %s in path %s: %v", method.Key(), pathKey, err)
			}
			tagsMatched++
			clausesMatched += len(decodedTag.Policies)

			// HTTP methods are matched case-sensitively downstream
			// Whatever case method.Key() returns from the parsed spec, normalize it to lowercase here.
			methodKey := strings.ToLower(method.Key())
			if methodKey != method.Key() {
				slog.Debug("Normalized method key to lowercase",
					"path", pathKey, "rawMethod", method.Key(), "method", methodKey)
			}
			for i := range decodedTag.Policies {
				slog.Debug("x-policies clause decoded from method-level tag",
					"path", pathKey, "method", methodKey, "index", i,
					"clauses", len(decodedTag.Policies), "clauseTypes", clauseTypes(&decodedTag.Policies[i]))
			}
			slog.Debug("Found x-policies tag on method",
				"path", pathKey, "method", methodKey, "rawMethod", method.Key(),
				"line", methodTagNode.Line, "clauses", len(decodedTag.Policies),
				"description", decodedTag.Description)

			var pathPolicies policy.PathPolicies
			if existing, ok := result.SpecializedPaths[pathKey]; ok {
				pathPolicies = existing
			} else {
				pathPolicies = policy.PathPolicies{
					Policies:           []policy.PolicyClause{},
					Path:               pathKey,
					SpecializedMethods: make(map[string]policy.PathMethodPolicies),
				}
			}
			if pathPolicies.SpecializedMethods == nil {
				pathPolicies.SpecializedMethods = make(map[string]policy.PathMethodPolicies)
			}
			pathPolicies.SpecializedMethods[methodKey] = policy.PathMethodPolicies{
				Policies: decodedTag.Policies,
				Path:     pathKey,
				Method:   methodKey,
			}
			result.SpecializedPaths[pathKey] = pathPolicies
		}
	}

	slog.Debug("Finished scanning the spec for x-policies tags",
		"pathsScanned", pathsScanned, "tagsMatched", tagsMatched,
		"clausesMatched", clausesMatched, "extensionsSeen", formatExtensionCounts(extensionCounts))
	if tagsMatched == 0 || clausesMatched == 0 {
		slog.Warn("No x-policies clauses found in the spec, the generated policy will deny every request",
			"pathsScanned", pathsScanned, "tagsMatched", tagsMatched,
			"clausesMatched", clausesMatched, "expectedExtension", XTeadalPoliciesKey,
			"extensionsSeen", formatExtensionCounts(extensionCounts))
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
		slog.Warn("Multiple security requirements found in OpenAPI spec, only bearerAuth is used")
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
		slog.Error("Failed to read the spec as an OpenAPI document", "error", err)
		return nil, fmt.Errorf("failed to parse OpenAPI spec: %v", err)
	}
	docModel, errors := document.BuildV3Model()
	if len(errors) > 0 {
		for i := range errors {
			slog.Error("Cannot build the OpenAPI v3 model", "index", i, "error", errors[i])
		}
		slog.Error("Cannot build the OpenAPI v3 model, aborting", "errorsReported", len(errors))
		panic(fmt.Sprintf("cannot create v3 model from document: %d errors reported", len(errors)))
	}
	return docModel, nil
}
