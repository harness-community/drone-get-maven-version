# Introducing the drone-get-maven-version Plugin

At Harness, we are dedicated to enhancing Continuous Integration (CI) and Continuous Deployment (CD) processes by providing tools that simplify complex workflows. We understand the significance of seamlessly integrating Maven builds with Docker image creation. That's why we are thrilled to introduce the **drone-get-maven-version** plugin. This plugin streamlines the process of accessing the POM Version, enabling you to effortlessly manage versioning and image creation in your CI/CD pipelines.

### What is the drone-get-maven-version plugin?

The **drone-get-maven-version** plugin is a versatile tool designed to simplify the integration of Maven builds with Docker image creation. This plugin automates the process of fetching the POM version in your maven project.

### Build the Docker Image

Using the plugin is straightforward. You can run the script directly using the following command:

    PLUGIN_POM_PATH=POM_PATH \
    go run main.go

Additionally, you can build the Docker image with these commands:

    docker buildx build -t DOCKER_ORG/drone-get-maven-version --platform linux/amd64 .

### Usage in Harness CI

Integrating the drone-get-maven-version Plugin into your Harness CI pipeline is seamless. You can use Docker to run the plugin with environment variables. Here's how:

    docker run --rm \
    -e PLUGIN_POM_PATH=${POM_PATH} \
    -v $(pwd):$(pwd) \
    -w $(pwd) \
    harnesscommunity/drone-get-maven-version

In your Harness CI pipeline, you can define the plugin as a step, like this:

    - step:
        type:  Plugin
        name:  drone-get-maven-version-plugin
        identifier:  maven_plugin
        spec:
            connectorRef:  docker-registry-connector
            image:  harnesscommunity/drone-get-maven-version
            settings:
                pom_path:  path-to-your-maven-project

### Plugin Options

The drone-get-maven-version plugin offers the following customization option:

- **pom_path**: The path to your Maven project. You should replace ${POM_PATH} with the actual path to your Maven project.

This environment variable is crucial for configuring and customizing the behavior of the drone-get-maven-version plugin when executed as a Docker container.

### Get Started with the drone-get-maven-version Plugin

Whether you are an experienced DevOps professional or new to CI/CD, the drone-get-maven-version plugin can simplify your Docker image creation process. Give it a try and witness how it streamlines your CI/CD pipelines!

For more information, documentation, and updates, please visit our GitHub repository: [drone-get-maven-version](https://github.com/harness-community/drone-get-maven-version).
