package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/joho/godotenv"
)

func main() {
	pomPath := os.Getenv("PLUGIN_POM_PATH")

	if pomPath == "" {
		fmt.Println("POM Path is empty, exiting...")
		os.Exit(1)
	}

	pathSeparator := "/"

	fmt.Println("POM Path: ", pomPath)

	// cmd := exec.Command("mvn", "-f", fmt.Sprintf("%s/pom.xml", pomPath), "help:evaluate", "-Dexpression=project.version", "-q", "-DforceStdout")
	cmd := exec.Command("mvn", "-f", fmt.Sprintf("%s%s%s", pomPath, pathSeparator, "pom.xml"), "help:evaluate", "-Dexpression=project.version", "-q", "-DforceStdout")
	output, err := cmd.Output()

	// check if os is windows

	if err != nil {
		fmt.Println("Error: ", err.Error())
		os.Exit(1)
	}

	pomVersion := strings.TrimSpace(string(output))

	fmt.Println("POM Version: ", pomVersion)
	os.Setenv("POM_VERSION", pomVersion)

	outputFilePath := os.Getenv("DRONE_OUTPUT")
	key := "POM_VERSION"

	err = WritePluginOutputFile(outputFilePath, key, pomVersion)
	if err != nil {
		fmt.Println("Error writing POM version to file:", err)
		os.Exit(1)
	}

	fmt.Printf("%s=%s written to %s\n", key, pomVersion, outputFilePath)
}

func WritePluginOutputFile(outputFilePath, key, value string) error {
	output := map[string]string{
		key: value,
	}
	return godotenv.Write(output, outputFilePath)
}
