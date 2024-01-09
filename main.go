package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
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

	outputFile, err := os.OpenFile(os.Getenv("DRONE_OUTPUT"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Error opening output file:", err)
		os.Exit(1)
	}
	defer outputFile.Close()

	_, err = fmt.Fprintf(outputFile, "POM_VERSION=%s\n", pomVersion)
	if err != nil {
		fmt.Println("Error writing to output file:", err)
		os.Exit(1)
	}

	fmt.Println("POM version written to DRONE_OUTPUT.env file")
}
