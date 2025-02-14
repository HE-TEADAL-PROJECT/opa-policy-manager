package commands

import (
	"dspn-regogenerator/config"
	"dspn-regogenerator/utils"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var oidc_file = ""
var rbacdb_file = ""
var service_file = ""

var iam_provider = ""

func setupEnv(path string, service_name string) error {

	if !utils.DirectoryExists(config.Root_output_dir) {
		fmt.Printf("Directory %s does not exist, creating it...", config.Root_output_dir)
		if err := os.MkdirAll(config.Root_output_dir, os.ModePerm); err != nil {
			fmt.Println("Error creating directory: %v", err)
			return err
		}
	}

	if utils.DirectoryExists(path) {
		fmt.Println("Directory %s exists, removing contained files...", path)
		if err := utils.RemoveFilesInDirectory(path); err != nil {
			fmt.Println("Error removing files: %v", err)
			return err
		}
	} else {
		fmt.Printf("Directory %s does not exist, creating it...", path)
		if err := os.MkdirAll(path, os.ModePerm); err != nil {
			fmt.Println("Error creating directory: %v", err)
			return err
		}
	}

	oidc_file = path + "/oidc_" + service_name + ".rego"
	rbacdb_file = path + "/rbacdb_" + service_name + ".rego"
	service_file = path + "/service_" + service_name + ".rego"

	//duplicate oidc_template_file
	content, err := os.ReadFile(config.Oidc_template_file)
	if err != nil {
		fmt.Println("Error reading template file: %v\n", err)
		return err
	}
	//template = string(content)

	if err := os.WriteFile(oidc_file, content, 0644); err != nil {
		fmt.Println("Error duplicating template file: %v\n", err)
		return err
	}

	//duplicate rbacdb_template_file
	content, err = os.ReadFile(config.Rbacdb_template_file)
	if err != nil {
		fmt.Println("Error reading template file: %v\n", err)
		return err
	}
	//template = string(content)

	if err = os.WriteFile(rbacdb_file, content, 0644); err != nil {
		fmt.Println("Error duplicating template file: %v\n", err)
		return err
	}

	//duplicate service_template_file
	content, err = os.ReadFile(config.Service_template_file)
	if err != nil {
		fmt.Println("Error reading template file: %v\n", err)
		return err
	}
	//template = string(content)

	if err = os.WriteFile(service_file, content, 0644); err != nil {
		fmt.Println("Error duplicating template file: %v\n", err)
		return err
	}

	return nil
}

func replace_placeholder_oidc(iam_provider string, oidc_name string) error {

	replacements := map[string]string{
		"{{DNS_OR_IP}}": iam_provider,
		"{{OIDC_NAME}}": oidc_name,
	}

	if err := utils.ReplacePlaceholdersInFile(oidc_file, replacements); err != nil {
		fmt.Println("Error: %v\n", err)
		return err
	} else {
		fmt.Println("Placeholders replaced successfully. Output saved in", oidc_file)
		return err
	}

	return nil
}

func replace_placeholder_service(service_name string, oidc_name string) error {

	replacements := map[string]string{
		"{{SERVICE_NAME}}": service_name,
		"{{OIDC_NAME}}":    oidc_name,
	}

	if err := utils.ReplacePlaceholdersInFile(service_file, replacements); err != nil {
		fmt.Println("Error: %v\n", err)
		return err
	} else {
		fmt.Println("Placeholders replaced successfully. Output saved in", service_file)
		return err
	}

	return nil
}

func replace_placeholder_rbacdb(service_name string, roles []string, users []string, roles_permissions map[string][]interface{}, users_permissions map[string][]interface{}) error {

	users_mapping := make(map[string]string)
	var formatted_users []string
	var i = 1
	for _, s := range users {
		users_mapping["user"+strconv.Itoa(i)] = s
		formatted_users = append(formatted_users, "user"+strconv.Itoa(i)+" := \""+s+"\"")
		i = i + 1
	}

	roles_mapping := make(map[string]string)
	var formatted_roles []string
	i = 1
	for _, s := range roles {
		roles_mapping["role"+strconv.Itoa(i)] = s
		formatted_roles = append(formatted_roles, "role"+strconv.Itoa(i)+" := \""+s+"\"")
		i = i + 1
	}

	//generate the section of the users permission in rego
	formatted_users_permission := ""
	for user_key, user_name := range users_mapping {
		already_found := false
		for method := range users_permissions {
			//fmt.Println("analyze method " + method)
			method_info := strings.Split(method, "@")
			for user := range users_permissions[method] {

				//fmt.Println(users_permissions[method][user].(string) + " " + user_name)

				if users_permissions[method][user].(string) == user_name {
					if !already_found {
						formatted_users_permission = formatted_users_permission + "\t" + user_key + ": ["
						already_found = true
					}

					formatted_users_permission = formatted_users_permission + "\n \t { \n \t \t \"methods\": http." + utils.GetMethodType(method_info[1]) + " \n \t \t \"url_regex\": \"^" + method_info[0] + "/.*\"\n \t },"
				}

			}
		}
		if already_found {
			formatted_users_permission = formatted_users_permission + "\n \t],\n"
		}
	}

	//generate the section of the roles permission in rego
	formatted_roles_permission := ""
	for role_key, role_name := range roles_mapping {
		already_found := false
		for method := range roles_permissions {
			//fmt.Println("analyze method " + method)
			method_info := strings.Split(method, "@")
			for role := range roles_permissions[method] {
				//fmt.Println(users_permissions[method][user].(string) + " " + user_name)

				if roles_permissions[method][role].(string) == role_name {
					if !already_found {
						formatted_roles_permission = formatted_roles_permission + "\t" + role_key + ": ["
						already_found = true
					}

					formatted_roles_permission = formatted_roles_permission + "\n \t { \n \t \t \"methods\": http." + utils.GetMethodType(method_info[1]) + " \n \t \t \"url_regex\": \"^" + method_info[0] + "/.*\"\n \t },"
				}

			}
		}
		if already_found {
			formatted_roles_permission = formatted_roles_permission + "\n \t],\n"
		}
	}

	replacements := map[string]string{
		"{{SERVICE_NAME}}":      service_name,
		"{{ROLES}}":             strings.Join(formatted_roles, "\n"),
		"{{USERS}}":             strings.Join(formatted_users, "\n"),
		"{{ROLES_PERMISSIONS}}": formatted_roles_permission,
		"{{USERS_PERMISSIONS}}": formatted_users_permission,
	}

	if err := utils.ReplacePlaceholdersInFile(rbacdb_file, replacements); err != nil {
		fmt.Println("Error: %v\n", err)
		return err
	} else {
		fmt.Println("Placeholders replaced successfully. Output saved in", rbacdb_file)
		return nil
	}

}

func downloadFile(url string, dest string) error {
	// Create the file
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Check server response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	return err
}

func GenerateRegoFilesCmd(service_name string, openAPI_URL string) {
	/*if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <service_name> <openAPI_file> [policy_dir]")
		return
	}
	service_name = os.Args[1]
	fmt.Println(os.Args[1])
	openAPI_file := os.Args[2]
	fmt.Println(os.Args[2])
	policy_dir := ""
	if len(os.Args) == 2 {
		policy_dir = "output/" + os.Args[3]
	} else {
		policy_dir = "output/" + filepath.Base(openAPI_file)
	}
	fmt.Println(policy_dir)*/

	//READ relevant info from openapi

	var policy_dir = config.Root_output_dir + service_name

	var openAPI_file = config.Root_schema_dir + service_name + "-api.json"

	fmt.Println("Downloading file...")
	err := downloadFile(openAPI_URL, openAPI_file)
	if err != nil {
		fmt.Println("Error downloading file:", err)
		return
	}
	fmt.Println("File downloaded successfully:", openAPI_file)

	iam_provider_int, err := utils.ExtractItemsFromOpenAPI(openAPI_file, "x-teadal-IAM-provider")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	iam_provider, err = utils.ExtractServerName(iam_provider_int.(string))
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	users, users_permissions, err := utils.ExtractUsersPermissions(openAPI_file)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println(users)
	fmt.Println(users_permissions)

	roles, roles_permissions, err := utils.ExtractRolesPermissions(openAPI_file)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println(roles)
	fmt.Println(roles_permissions)

	// setup the environment duplicating the templates

	err = setupEnv(policy_dir, service_name)
	if err != nil {
		fmt.Println("Error in setting up the environment")
	}

	//replace the placeholders

	err = replace_placeholder_oidc(iam_provider, strings.TrimSuffix(filepath.Base(oidc_file), ".rego"))
	if err != nil {
		fmt.Println("Error in updating the oidc file")
	}

	err = replace_placeholder_service(service_name, strings.TrimSuffix(filepath.Base(oidc_file), ".rego"))
	if err != nil {
		fmt.Println("Error in updating the oidc file")
	}

	err = replace_placeholder_rbacdb(service_name, roles, users, roles_permissions, users_permissions)
	if err != nil {
		fmt.Println("Error in updating the oidc file")
	}

}
