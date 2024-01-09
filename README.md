# Introducing the drone-maven-docker-image-builder Plugin

At Harness, we are dedicated to enhancing Continuous Integration (CI) and Continuous Deployment (CD) processes by providing tools that simplify complex workflows. We understand the significance of seamlessly integrating Maven builds with Docker image creation. That's why we are thrilled to introduce the **drone-maven-docker-image-builder** plugin. This plugin streamlines the process of accessing the POM Version, enabling you to effortlessly manage versioning and image creation in your CI/CD pipelines.

### What is the drone-maven-docker-image-builder plugin?

The **drone-maven-docker-image-builder** plugin is a versatile tool designed to simplify the integration of Maven builds with Docker image creation. This plugin automates the process of fetching the POM version in your maven project.

### Build the Docker Image

Using the plugin is straightforward. You can run the script directly using the following command:

    PLUGIN_POM_PATH=POM_PATH \
    go run main.go

Additionally, you can build the Docker image with these commands:

    docker buildx build -t DOCKER_ORG/drone-maven-version-docker-build --platform linux/amd64 .

### Usage in Harness CI

Integrating the drone-maven-docker-image-builder Plugin into your Harness CI pipeline is seamless. You can use Docker to run the plugin with environment variables. Here's how:

    docker run --rm \
    -e PLUGIN_POM_PATH=${POM_PATH} \
    -v $(pwd):$(pwd) \
    -w $(pwd) \
    harnesscommunity/maven-docker-image-builder

In your Harness CI pipeline, you can define the plugin as a step, like this:

    - step:
        type:  Plugin
        name:  drone-maven-docker-image-builder-plugin
        identifier:  maven_plugin
        spec:
            connectorRef:  docker-registry-connector
            image:  harnesscommunity/drone-maven-docker-image-builder
            settings:
                pom_path:  path-to-your-maven-project

### Plugin Options

The Maven Docker Image Builder Plugin offers the following customization option:

- **pom_path**: The path to your Maven project. You should replace ${POM_PATH} with the actual path to your Maven project.

This environment variable is crucial for configuring and customizing the behavior of the drone-maven-docker-image-builder plugin when executed as a Docker container.

### Get Started with the Maven Docker Image Builder Plugin

Whether you are an experienced DevOps professional or new to CI/CD, the drone-maven-docker-image-builder plugin can simplify your Docker image creation process. Give it a try and witness how it streamlines your CI/CD pipelines!

For more information, documentation, and updates, please visit our GitHub repository: [drone-maven-docker-image-builder](https://github.com/harness-community/drone-maven-docker-image-builder).
