package commands

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"dspn-regogenerator/config"
	"dspn-regogenerator/utils"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// createBundle creates an OPA-compliant policy bundle
func createBundle(policyDir, outputFile string) error {
	// Create output file
	outFile, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create bundle file: %w", err)
	}
	defer outFile.Close()

	// Create gzip writer
	gzipWriter := gzip.NewWriter(outFile)
	defer gzipWriter.Close()

	// Create tar writer
	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	// Walk through the policy directory
	err = filepath.Walk(policyDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		// Open policy file
		file, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("failed to open policy file: %w", err)
		}
		defer file.Close()

		// Prepare tar header
		relPath, _ := filepath.Rel(policyDir, path)
		header := &tar.Header{
			Name: relPath,
			Size: info.Size(),
			Mode: int64(info.Mode()),
		}
		err = tarWriter.WriteHeader(header)
		if err != nil {
			return fmt.Errorf("failed to write tar header: %w", err)
		}

		// Copy file content to tar
		_, err = io.Copy(tarWriter, file)
		if err != nil {
			return fmt.Errorf("failed to write file to bundle: %w", err)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to create bundle: %w", err)
	}

	fmt.Println("Bundle created successfully at", outputFile)
	return nil
}

func containsString(arr []string, target string) bool {
	for _, v := range arr {
		if v == target {
			return true
		}
	}
	return false
}

func getServiceList() ([]string, error) {

	//download existing bundle
	fmt.Println("Downloading existing bundle..." + "http://" + config.Config.Minio_Server + "/" + config.Config.Bucket_Name + "/" + config.Config.BundleFileName)
	if err := utils.DownloadFile("http://"+config.Config.Minio_Server+"/"+config.Config.Bucket_Name+"/"+config.Config.BundleFileName, config.Root_bundle_dir+"/"+config.Config.BundleFileName); err != nil {
		fmt.Println("Error:", err)
		return nil, err
	}
	fmt.Println("Download completed successfully!")

	return listDirectoriesInTarGz(config.Root_bundle_dir+"/"+config.Config.BundleFileName, "rego")

}

func listDirectoriesInTarGz(filename string, mainDir string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)
	dirSet := make(map[string]struct{})

	std_dir := []string{"authnz", "config", "main.rego", "data.json"}
	result := []string{}

	for {
		header, err := tarReader.Next()
		if err != nil {
			break
		}

		//if header.Typeflag == tar.TypeDir {
		//if strings.HasPrefix(header.Name, mainDir+"/") {
		fmt.Println("check" + header.Name)
		dirName := strings.TrimPrefix(header.Name, mainDir+"/")
		if dirName == header.Name {
			dirName = strings.TrimPrefix(header.Name, "/"+mainDir+"/")
		}
		dirName = strings.Split(dirName, "/")[0]
		if dirName != "" {
			dirSet[dirName] = struct{}{}
		}
		//dirSet[header.Name] = struct{}{}

		//}
		//}
	}

	fmt.Println("Directories under", mainDir, ":")
	for dir := range dirSet {
		fmt.Println("- " + dir)
		if !containsString(std_dir, dir) {
			fmt.Println("included")
			result = append(result, dir)
		}
	}

	return result, nil
}

func replace_placeholder_rego(mainrego_file string, new_import string, new_allow string) error {

	replacements := map[string]string{
		"{{IMPORT}}": new_import,
		"{{ALLOW}}":  new_allow,
	}

	if err := utils.ReplacePlaceholdersInFile(mainrego_file, replacements); err != nil {
		fmt.Println("Error: %v\n", err)
		return err
	}
	fmt.Println("Placeholders replaced successfully. Output saved in", mainrego_file)
	return nil
}

func GenerateBundleCmd(serviceName string) error {

	var policyDir = config.Root_output_dir + serviceName

	ctx := context.Background()

	//download existing bundle
	fmt.Println("Downloading existing bundle..." + "http://" + config.Config.Minio_Server + "/" + config.Config.Bucket_Name + "/" + config.Config.BundleFileName)
	if err := utils.DownloadFile("http://"+config.Config.Minio_Server+"/"+config.Config.Bucket_Name+"/"+config.Config.BundleFileName, config.Root_bundle_dir+"/"+config.Config.BundleFileName); err != nil {
		fmt.Println("Error:", err)
		return err
	}
	fmt.Println("Download completed successfully!")

	//extract files from bundle
	tempDir := config.Root_bundle_dir + "/" + config.Config.BundleName
	os.MkdirAll(tempDir, os.ModePerm)

	fmt.Println("Extracting existing bundle in " + tempDir + "...")
	if err := utils.ExtractTarGz(config.Root_bundle_dir+"/"+config.Config.BundleFileName, tempDir); err != nil {
		fmt.Println("Error extracting bundle:", err)
		return err
	}

	//upload the old bucket with a new name
	timestamp := time.Now().Format("20060102_1504")
	newBundleFileName := fmt.Sprintf("%s_%s", strings.TrimSuffix(config.Config.BundleFileName, "-LATEST.tar.gz"), timestamp) + ".tar.gz"

	if err := os.Rename(config.Root_bundle_dir+"/"+config.Config.BundleFileName, config.Root_bundle_dir+"/"+newBundleFileName); err != nil {
		fmt.Println("Error renaming file:", err)
		return err
	}

	fmt.Println("File renamed successfully from", config.Root_bundle_dir+"/"+config.Config.BundleFileName, "to", config.Root_bundle_dir+"/"+newBundleFileName)

	minioClient, err := minio.New(config.Config.Minio_Server, &minio.Options{
		Creds:  credentials.NewStaticV4(config.Config.Minio_Access_Key, config.Config.Minio_Secret_Key, ""),
		Secure: false,
	})
	if err != nil {
		log.Fatalln("Failed to initialize MinIO client:", err)
		return err
	}

	info, err := minioClient.FPutObject(ctx, config.Config.Bucket_Name, newBundleFileName, config.Root_bundle_dir+"/"+newBundleFileName, minio.PutObjectOptions{ContentType: "application/x-gzip"})
	if err != nil {
		log.Fatalln(err)
		return err
	}

	log.Printf("Successfully uploaded %s of size %d\n", newBundleFileName, info.Size)

	//mergin the polciies
	fmt.Println("Merging new policies...")
	if err := utils.MoveDirectory(policyDir, config.Root_bundle_dir+"/"+config.Config.BundleName+"/rego"); err != nil {
		fmt.Println("Error:", err)
		return err
	}

	fmt.Println("Successfully moved", policyDir, "to", config.Root_bundle_dir+"/"+config.Config.BundleName+"/rego")

	//duplicate main_template_file
	main_file := config.Root_bundle_dir + "/" + config.Config.BundleName + "/rego/main.rego"

	content, err := os.ReadFile(config.Main_template_file)
	if err != nil {
		fmt.Println("Error reading template file: %v\n", err)
		return err
	}
	if err := os.WriteFile(main_file, content, 0644); err != nil {
		fmt.Println("Error duplicating template file: %v\n", err)
		return err
	}

	new_import := ""
	new_allow := ""
	//get list of services
	if serviceList, err := getServiceList(); err != nil {
		fmt.Println("Error", err)
		return err
	} else {
		fmt.Println("List of registered services with policies")
		for service := range serviceList {
			new_import = new_import + "\n" + "import data." + serviceList[service] + ".service as " + serviceList[service]
			new_allow = new_allow + "\n\nallow {\n\t " + serviceList[service] + ".allow\n}"

		}
		new_import = new_import + "\n" + "import data." + serviceName + ".service as " + serviceName
		new_allow = new_allow + "\n\nallow {\n\t " + serviceName + ".allow\n}"
	}

	if err = replace_placeholder_rego(config.Root_bundle_dir+"/"+config.Config.BundleName+"/rego/main.rego", new_import, new_allow); err != nil {
		fmt.Println("Error:", err)
		return err
	}

	//create the new bundle

	if err = createBundle(config.Root_bundle_dir+"/"+config.Config.BundleName, config.Root_bundle_dir+"/"+config.Config.BundleFileName); err != nil {
		fmt.Println("Error:", err)
		return err
	}

	info, err = minioClient.FPutObject(ctx, config.Config.Bucket_Name, config.Config.BundleFileName, config.Root_bundle_dir+"/"+config.Config.BundleFileName, minio.PutObjectOptions{ContentType: "application/x-gzip"})
	if err != nil {
		log.Fatalln(err)
		return err
	}
	return nil

}

func ListServicePolicies() ([]string, error) {

	return getServiceList()

}

func DeleteServicePolicies(service_name string) error {

	ctx := context.Background()

	if serviceList, err := getServiceList(); err != nil {
		fmt.Println("Error", err)
		return err
	} else {
		if !containsString(serviceList, service_name) {

			fmt.Println("Service "+service_name+" is not registered", err)
			return err
		} else {
			fmt.Println("Service " + service_name + " found in the bundle")

			//download existing bundle
			fmt.Println("Downloading existing bundle..." + "http://" + config.Config.Minio_Server + "/" + config.Config.Bucket_Name + "/" + config.Config.BundleFileName)
			if err := utils.DownloadFile("http://"+config.Config.Minio_Server+"/"+config.Config.Bucket_Name+"/"+config.Config.BundleFileName, config.Root_bundle_dir+"/"+config.Config.BundleFileName); err != nil {
				fmt.Println("Error:", err)
				return err
			}
			fmt.Println("Download completed successfully!")

			//extract files from bundle
			tempDir := config.Root_bundle_dir + "/" + config.Config.BundleName
			if err := os.MkdirAll(tempDir, os.ModePerm); err != nil {
				fmt.Println("Error creating temp dir:", err)
				return err
			}

			fmt.Println("Extracting existing bundle in " + tempDir + "...")
			if err := utils.ExtractTarGz(config.Root_bundle_dir+"/"+config.Config.BundleFileName, tempDir); err != nil {
				fmt.Println("Error extracting bundle:", err)
				return err
			}

			//upload the old bucket with a new name
			timestamp := time.Now().Format("20060102_1504")
			newBundleFileName := fmt.Sprintf("%s_%s", strings.TrimSuffix(config.Config.BundleFileName, "-LATEST.tar.gz"), timestamp) + ".tar.gz"

			if err := os.Rename(config.Root_bundle_dir+"/"+config.Config.BundleFileName, config.Root_bundle_dir+"/"+newBundleFileName); err != nil {
				fmt.Println("Error renaming file:", err)
				return err
			}

			fmt.Println("File renamed successfully from", config.Root_bundle_dir+"/"+config.Config.BundleFileName, "to", config.Root_bundle_dir+"/"+newBundleFileName)

			minioClient, err := minio.New(config.Config.Minio_Server, &minio.Options{
				Creds:  credentials.NewStaticV4(config.Config.Minio_Access_Key, config.Config.Minio_Secret_Key, ""),
				Secure: false,
			})
			if err != nil {
				log.Fatalln("Failed to initialize MinIO client:", err)
				return err
			}

			info, err := minioClient.FPutObject(ctx, config.Config.Bucket_Name, newBundleFileName, config.Root_bundle_dir+"/"+newBundleFileName, minio.PutObjectOptions{ContentType: "application/x-gzip"})
			if err != nil {
				log.Fatalln(err)
				return err
			}

			log.Printf("Successfully uploaded %s of size %d\n", newBundleFileName, info.Size)

			//remove dir
			os.RemoveAll(config.Root_bundle_dir + "/" + config.Config.BundleName + "/rego/" + service_name)

			// duplicate main_template_file
			main_file := config.Root_bundle_dir + "/" + config.Config.BundleName + "/rego/main.rego"

			content, err := os.ReadFile(config.Main_template_file)
			if err != nil {
				fmt.Println("Error reading template file: %v\n", err)
				return err
			}
			if err := os.WriteFile(main_file, content, 0644); err != nil {
				fmt.Println("Error duplicating template file: %v\n", err)
				return err
			}

			new_import := ""
			new_allow := ""
			// get list of services
			if serviceList, err := getServiceList(); err != nil {
				fmt.Println("Error", err)
				return err
			} else {
				fmt.Println("List of registered services with policies")
				for service := range serviceList {
					if serviceList[service] != service_name {
						new_import = new_import + "\n" + "import data." + serviceList[service] + ".service as " + serviceList[service]
						new_allow = new_allow + "\n\nallow {\n\t " + serviceList[service] + ".allow\n}"
						fmt.Println(new_import)
					}
				}
			}

			if err = replace_placeholder_rego(config.Root_bundle_dir+"/"+config.Config.BundleName+"/rego/main.rego", new_import, new_allow); err != nil {
				fmt.Println("Error:", err)
				return err
			}
			if err = createBundle(config.Root_bundle_dir+"/"+config.Config.BundleName, config.Root_bundle_dir+"/"+config.Config.BundleFileName); err != nil {
				fmt.Println("Error:", err)
				return err
			}

			info, err = minioClient.FPutObject(ctx, config.Config.Bucket_Name, config.Config.BundleFileName, config.Root_bundle_dir+"/"+config.Config.BundleFileName, minio.PutObjectOptions{ContentType: "application/x-gzip"})
			if err != nil {
				log.Fatalln(err)
				return err
			} else {
				fmt.Print(info)
			}
			return nil
		}

	}

}
