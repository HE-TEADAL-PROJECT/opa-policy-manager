package config

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type ConfigType struct {
	Minio_Server     string ""
	Minio_Access_Key string ""
	Minio_Secret_Key string ""
	Bucket_Name      string ""
	BundleName       string ""
	BundleFileName   string ""
}

//var basicBundleName = "teadal-policy-bundle-LATEST"
//var basicBundleFilename = basicBundleName + ".tar.gz"

//var bundle_dir = "./bundles"

var Config ConfigType

var config_file = "opa-policy-manager.config"

func TestMinio() error {

	minioClient, err := minio.New(Config.Minio_Server, &minio.Options{
		Creds:  credentials.NewStaticV4(Config.Minio_Access_Key, Config.Minio_Secret_Key, ""),
		Secure: false,
	})
	if err != nil {
		return fmt.Errorf("Failed to initialize MinIO client:", err)

	} else {
		fmt.Println("Minio server found and connected")
		exist, err := minioClient.BucketExists(context.Background(), Config.Bucket_Name)
		if err != nil {
			return fmt.Errorf("Failed to find the policy bucket '%s' %w", Config.Bucket_Name, err)
		} else {
			if exist {
				fmt.Println("Bucket " + Config.Bucket_Name + " exists")
				fmt.Println("Minio Test passed")
				return nil
			} else {
				return fmt.Errorf("Failed to find the policy bucket '%s' %w", Config.Bucket_Name, err)
			}

		}

	}
}

func SaveConfigToFile() error {

	file, err := os.Create(config_file)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	return encoder.Encode(Config)
}

func LoadConfigFromFile() error {
	file, err := os.Open(config_file)
	if err != nil {
		fmt.Println("opa-policy-manager.config does not exist. Read initial values from env variables" + os.Getenv("BUCKET_NAME"))
		Config.Minio_Server = os.Getenv("MINIO_SERVER")
		Config.Minio_Access_Key = os.Getenv("MINIO_ACCESS_KEY")
		Config.Minio_Secret_Key = os.Getenv("MINIO_SECRET_KEY")
		Config.Bucket_Name = os.Getenv("BUCKET_NAME")
		Config.BundleName = os.Getenv("BUNDLE_NAME")
		Config.BundleFileName = os.Getenv("BUNDLE_NAME") + ".tar.gz"
		SaveConfigToFile()

		return nil
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	return decoder.Decode(&Config)
}
