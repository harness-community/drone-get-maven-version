# drone-push-helm-chart-docker-registry

- [Synopsis](#Synopsis)
- [Parameters](#Paramaters)
- [Plugin Image](#Plugin-Image)
- [Examples](#Examples)

## Synopsis

This plugin is designed to streamline the packaging and distribution of Helm charts to a Docker registry.

To learn how to utilize Drone plugins in Harness CI, please consult the provided [documentation](https://developer.harness.io/docs/continuous-integration/use-ci/use-drone-plugins/run-a-drone-plugin-in-ci).

## Parameters

| Parameter                                                                                                                      | Choices/<span style="color:blue;">Defaults</span>                  | Comments                                                   |
| :----------------------------------------------------------------------------------------------------------------------------- | :----------------------------------------------------------------- | :--------------------------------------------------------- |
| chart_name <span style="font-size: 10px"><br/>`string`</span> <span style="color:red; font-size: 10px">`required`</span>       |                                                                    | The name of the chart in Chart.yaml                        |
| chart_version <span style="font-size: 10px"><br/>`string`</span>                                                               | Defaults: <span style="color:blue;">1.0.0</span>                   | The project version present in Chart.yaml                  |
| docker_registry <span style="font-size: 10px"><br/>`string`</span>                                                             | Defaults: <span style="color:blue;">registry.hub.docker.com</span> | Docker registry where the packaged chart will be published |
| chart_path <span style="font-size: 10px"><br/>`string`</span>                                                                  | Defaults: <span style="color:blue;">`./`</span>                    | Directory containing the helm chart                        |
| docker_username <span style="font-size: 10px"><br/>`string`</span> <span style="color:red; font-size: 10px">`required`</span>  |                                                                    | Docker username to login to the above registry.            |
| docker_password <span style="font-size: 10px"><br/>`string`</span> <span style="color:red; font-size: 10px">`required`</span>  |                                                                    | Docker PAT to authenticate                                 |
| docker_namespace <span style="font-size: 10px"><br/>`string`</span> <span style="color:red; font-size: 10px">`required`</span> |                                                                    | Namespace under which the chart will be published          |

## Plugin Image

The plugin `harnesscommunity/drone-push-helm-chart-docker-registry` is available for the following architectures:

| OS            | Tag             |
| ------------- | --------------- |
| linux/amd64   | `linux-amd64`   |
| linux/arm64   | `linux-arm64`   |
| windows/amd64 | `windows-amd64` |

## Examples

```
# Plugin YAML
- step:
    type: Plugin
    name: Push Helm to Docker
    identifier: Push_Helm_to_Docker
    spec:
        connectorRef: harness-docker-connector
        image: harnesscommunity/drone-helm-chart-docker-registry:linux-amd64
        settings:
            chart_name: mywebapp
            docker_username: <+variable.docker_username>
            docker_password: <+secrets.getValue("docker_pat")>
            chart_path: test
            chart_version: 5.0.0
            docker_namespace: <+variable.namespace>
```

> <span style="font-size: 14px; margin-left:5px; background-color: #d3d3d3; padding: 4px; border-radius: 4px;">ℹ️ If you notice any issues in this documentation, you can [edit this document](https://github.com/harness-community/drone-push-helm-chart-docker-registry/blob/main/README.md) to improve it.</span>
