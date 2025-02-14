package utils

import (
	"encoding/json"
	"fmt"
	"os"
)

type OpenAPIDocument map[string]interface{}

func ParseOpenAPIFile(filename string) (OpenAPIDocument, error) {
	file, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %v", err)
	}

	var data OpenAPIDocument
	if err := json.Unmarshal(file, &data); err != nil {
		return nil, fmt.Errorf("error parsing OpenAPI JSON: %v", err)
	}

	return data, nil
}

func GetOpenAPIField(data OpenAPIDocument, key string) (interface{}, error) {
	for k, v := range data {
		if k == key {
			return v, nil
		}
		if nestedMap, ok := v.(map[string]interface{}); ok {
			if result, err := GetOpenAPIField(nestedMap, key); err == nil {
				return result, nil
			}
		}
	}
	return nil, fmt.Errorf("key '%s' not found in OpenAPI document", key)
}

func ExtractItemsFromOpenAPI(filename string, key string) (interface{}, error) {

	doc, err := ParseOpenAPIFile(filename)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return nil, err
	}
	//fmt.Println("looking for " + key)
	value, err := GetOpenAPIField(doc, key)

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return nil, err
	}
	return value, nil

}

func GetPaths(data OpenAPIDocument) (map[string]interface{}, error) {
	paths, exists := data["paths"].(map[string]interface{})
	if !exists {
		return nil, fmt.Errorf("'paths' field not found in OpenAPI document")
	}
	return paths, nil
}

func ExtractUsersPermissions(filename string) ([]string, map[string][]interface{}, error) {
	doc, err := ParseOpenAPIFile(filename)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return nil, nil, err
	}
	paths, err := GetPaths(doc)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return nil, nil, err
	}

	users_permissions_list := make(map[string][]interface{}, 0)

	for path:= range paths {
		
		method := paths[path].(map[string]interface{})
		for method_name, method_info := range method {
			
			users, err := GetOpenAPIField(method_info.(map[string]interface{}), "x-teadal-users-allowed")
			if err == nil {				
				users_permissions_list[path + "@" + method_name] = users.([]interface{})
			}
		}
	}
	return UniqueElements(users_permissions_list), users_permissions_list, nil

}

func ExtractRolesPermissions(filename string) ([]string, map[string][]interface{}, error) {
	doc, err := ParseOpenAPIFile(filename)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return nil, nil, err
	}
	paths, err := GetPaths(doc)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return nil, nil, err
	}

	roles_permissions_list := make(map[string][]interface{}, 0)

	for path := range paths {
		method := paths[path].(map[string]interface{})
		for method_name, method_info := range method {
			
			roles, err := GetOpenAPIField(method_info.(map[string]interface{}), "x-teadal-roles-allowed")
			if err == nil {
				roles_permissions_list[path + "@" + method_name] = roles.([]interface{})
			}
		}

		

	}
	return UniqueElements(roles_permissions_list), roles_permissions_list, nil

}
