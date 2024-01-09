# drone-get-maven-version

- [Synopsis](#Synopsis)
- [Parameters](#Paramaters)
- [Notes](#Notes)
- [Plugin Image](#Plugin-Image)
- [Examples](#Examples)

## Synopsis

The plugin is designed to retrieve the `POM_VERSION` from the `pom.xml` file and assign it as an environment variable. This variable is then available for use in subsequent stages of the pipeline.

To learn how to utilize Drone plugins in Harness CI, please consult the provided [documentation](https://developer.harness.io/docs/continuous-integration/use-ci/use-drone-plugins/run-a-drone-plugin-in-ci).

## Parameters

| Parameter                                                                                                              | Choices/<span style="color:blue;">Defaults</span> | Comments                                             |
| :--------------------------------------------------------------------------------------------------------------------- | :------------------------------------------------ | ---------------------------------------------------- |
| pom_path <span style="font-size: 10px"><br/>`string`</span> <span style="color:red; font-size: 10px">`required`</span> |                                                   | Path to the directory containing the `pom.xml` file. |

## Notes

The plugin generates an output of the `POM_VERSION` as an environment variable. This variable is accessible in subsequent stages of the pipeline using the notation: `<+steps.STEP_ID.output.outputVariables.POM_VERSION>`

## Plugin Image

The plugin `harnesscommunity/drone-get-maven-version` is available for the following architectures:

| OS            | Tag             |
| ------------- | :-------------- |
| linux/amd64   | `linux-amd64`   |
| linux/arm64   | `linux-arm64`   |
| windows/amd64 | `windows-amd64` |

## Examples

```
# Plugin YAML
- step:
    type: Plugin
    name: drone-get-maven-version-plugin
    identifier: maven_plugin
    spec:
        connectorRef: harness-docker-connector
        image: harnesscommunity/drone-get-maven-version:linux-amd64
        settings:
            POM_PATH: .

# Build and push the docker image with POM version as the tag:
-step:
    type: BuildAndPushDockerRegistry
    name: BuildAndPushDockerRegistry
    identifier: BuildAndPushDockerRegistry
    spec:
        connectorRef: harness-docker-connector
        repo: namespace/container-name
        tags:
            - <+steps.STEP_ID.output.outputVariables.POM_VERSION>
```

> <span style="font-size: 14px; margin-left:5px; background-color: #d3d3d3; padding: 4px; border-radius: 4px;">ℹ️ If you notice any issues in this documentation, you can [edit this document](https://github.com/harness-community/drone-get-maven-version/blob/main/README.md) to improve it.</span>
