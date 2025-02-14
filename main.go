package main

import (
	"dspn-regogenerator/commands"
	"dspn-regogenerator/config"
	"fmt"
	"net/http"
	"os"
	"slices"
	"sync"

	"github.com/gin-gonic/gin"
)

type Policy struct {
	ID          string `json:"id"`
	ServiceName string `json:"serviceName"`
	OpenAPISpec string `json:"openAPIspec"`
}

var (
	mutex sync.Mutex
)

func getConfig(c *gin.Context) {
	c.JSON(http.StatusOK, config.Config)
}

func setConfig(c *gin.Context) {
	var newConfig config.ConfigType
	if err := c.ShouldBindJSON(&newConfig); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}
	mutex.Lock()
	config.Config = newConfig
	err := config.TestMinio()
	if err != nil {
		fmt.Errorf("Cannot connect to the Minio server ")
	} else {
		config.SaveConfigToFile()
	}
	mutex.Unlock()
	c.Status(http.StatusOK)
}

func getPolicies(c *gin.Context) {
	mutex.Lock()
	//serviceList := []string{}
	serviceList, err := commands.ListServicePolicies()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	} else {

		mutex.Unlock()
		c.JSON(http.StatusOK, serviceList)
	}
}

func addPolicy(c *gin.Context) {
	var newPolicy Policy
	if err := c.ShouldBindJSON(&newPolicy); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}
	if newPolicy.ServiceName == "" || newPolicy.OpenAPISpec == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing required parameters: servicename and openAPIspec"})
		return
	}
	mutex.Lock()
	commands.GenerateRegoFilesCmd(newPolicy.ServiceName, newPolicy.OpenAPISpec)
	commands.GenerateBundleCmd(newPolicy.ServiceName)
	mutex.Unlock()
	c.Status(http.StatusOK)
}

func deletePolicy(c *gin.Context) {
	id := c.Param("id")
	mutex.Lock()

	serviceList, err := commands.ListServicePolicies()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	} else {
		if !slices.Contains(serviceList, id) {
			mutex.Unlock()
			c.JSON(http.StatusNotFound, gin.H{"error": "Policy not found"})
			return
		}

		commands.DeleteServicePolicies(id)
		mutex.Unlock()
		c.Status(http.StatusOK)
	}
}

func main() {

	config.LoadConfigFromFile()
	fmt.Println(config.Config)
	os.MkdirAll(config.Root_bundle_dir, os.ModePerm)
	os.MkdirAll(config.Root_output_dir, os.ModePerm)

	r := gin.Default()

	r.GET("/config", getConfig)
	r.POST("/config", setConfig)

	r.GET("/policies", getPolicies)
	r.POST("/policies", addPolicy)
	r.DELETE("/policies/:id", deletePolicy)

	fmt.Println("Server running on port 8080...")
	r.Run(":8080")
}
