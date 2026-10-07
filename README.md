# drone-get-maven-version

- [Synopsis](#Synopsis)
- [Parameters](#Parameters)
- [Outputs](#Outputs)
- [Plugin Image](#Plugin-Image)
- [Examples](#Examples)
- [Development](#Development)

## Synopsis

The plugin reads Maven coordinates from a `pom.xml` and publishes them as Harness step output variables, available to later steps and stages.

It has two modes:

- `effective` (default): runs `mvn help:evaluate -Dexpression=project.version` and outputs `POM_VERSION`, the effective version after Maven resolves parents, properties, and `-D` values. This is the plugin's original behavior, unchanged.
- `raw_gav`: parses the POM file directly, without Maven, Java, or network access, and outputs the project's `groupId`, `artifactId`, and `version` under a configurable prefix. This replaces the Bamboo `maven-pom-parser-plugin` task (`pomFile` + `variablePrefix`).

To learn how to utilize Drone plugins in Harness CI, please consult the provided [documentation](https://developer.harness.io/docs/continuous-integration/use-ci/use-drone-plugins/run-a-drone-plugin-in-ci).

## Parameters

| Parameter | Choices/<span style="color:blue;">Defaults</span> | Comments |
| :-- | :-- | :-- |
| pom_path <span style="font-size: 10px"><br/>`string`</span> | | Path to the directory containing the `pom.xml` file. One of `pom_path` or `pom_file` is required. |
| pom_file <span style="font-size: 10px"><br/>`string`</span> | | Path to a POM file with any file name, for example `pom.xml` or `build/parent-pom.xml`. Takes precedence over `pom_path` (a warning is printed when both are set). |
| mode <span style="font-size: 10px"><br/>`string`</span> | `effective`, `raw_gav`<br/><span style="color:blue;">`effective`</span> | `effective` asks Maven for the effective version. `raw_gav` reads the POM file without Maven. |
| variable_prefix <span style="font-size: 10px"><br/>`string`</span> | <span style="color:blue;">`MAVEN`</span> | Prefix for the `raw_gav` output names. It is uppercased and every run of characters other than letters and digits becomes `_`, so `maven` becomes `MAVEN` and `my-app` becomes `MY_APP`. Ignored in `effective` mode. |

Relative paths are resolved against the step's working directory (the stage workspace). Both `/` and `\` separators are accepted, and paths may contain spaces.

### `raw_gav` rules

- Only direct children of `<project>` are read. Values inside `<dependencies>`, `<dependencyManagement>`, `<build>`, `<profiles>` and similar sections are never used.
- When `<groupId>` or `<version>` is absent, the value from `<parent>` is used and a log line says so. `<artifactId>` is never inherited.
- `${name}` placeholders are replaced from the POM's own `<properties>` and from `project.groupId`, `project.artifactId`, `project.version`, `project.parent.*` and `parent.*`. Parent POMs, `settings.xml`, environment variables and `-D` arguments are not read. A placeholder that cannot be resolved this way (for example a CI-friendly `${revision}` set on the command line) fails the step; use `mode: effective` for those projects.
- A missing or empty `groupId`, `artifactId` or `version` fails the step and names the field.
- UTF-8 (with or without BOM), US-ASCII and ISO-8859-1 POMs are supported, with LF or CRLF line endings.

## Outputs

| Mode | Outputs |
| :-- | :-- |
| `effective` | `POM_VERSION` |
| `raw_gav` | `<PREFIX>_GROUP_ID`, `<PREFIX>_ARTIFACT_ID`, `<PREFIX>_VERSION`, and `POM_VERSION` (same value as `<PREFIX>_VERSION`) |

Reference an output in a later step with `<+steps.STEP_ID.output.outputVariables.NAME>`, for example `<+steps.read_pom.output.outputVariables.MAVEN_VERSION>`.

The step fails with a non-zero exit code and an error on stderr when the POM cannot be read or parsed, a value is missing or unresolved, Maven fails (Maven's output is included), or a value would contain a line break.

## Plugin Image

The plugin `harnesscommunity/drone-get-maven-version` is available for the following platforms:

| OS | Tag | Notes |
| :-- | :-- | :-- |
| linux/amd64 | `linux-amd64` | |
| linux/arm64 | `linux-arm64` | |
| windows/amd64, Windows Server 2019 | `windows-ltsc2019`, `windows-ltsc2019-rN` | |
| windows/amd64, Windows Server 2022 | `windows-ltsc2022`, `windows-ltsc2022-rN` | |
| windows/amd64, Windows Server 2025 | `windows-ltsc2025`, `windows-ltsc2025-rN` | |
| windows/amd64, Windows Server 2022 | `windows-amd64` | Deprecated. Kept for existing pipelines; use `windows-ltsc2022`. |

Windows images:

- Pick the tag that matches the Windows build of the node or VM that runs the step (2019: 17763, 2022: 20348, 2025: 26100). Windows containers with process isolation require that match.
- `-rN` tags (`windows-ltsc2022-r1`, `-r2`, ...) are immutable releases and are never overwritten. Pin one for reproducible pipelines. The tag without `-rN` moves to the latest promoted release of that LTSC.
- The images are built on `harness/ci-base` for the matching LTSC, pinned by digest, and include Git from the base. Maven and a Temurin 17 JRE are installed for `effective` mode (`JAVA_HOME=C:\tools\java`, `MAVEN_HOME=C:\tools\maven`). `raw_gav` uses neither.
- `effective` mode downloads `maven-help-plugin` from your configured Maven repositories at run time, as before. `raw_gav` needs no network access.

## Examples

```yaml
# Read groupId, artifactId and version from the POM (Bamboo maven-pom-parser-plugin replacement)
- step:
    type: Plugin
    name: Read Maven POM values
    identifier: read_pom
    spec:
      connectorRef: harness-docker-connector
      image: harnesscommunity/drone-get-maven-version:windows-ltsc2022
      settings:
        pom_file: pom.xml
        variable_prefix: maven
        mode: raw_gav

# Use the values in a later step
- step:
    type: Run
    name: Show coordinates
    identifier: show_coordinates
    spec:
      shell: Powershell
      command: |-
        Write-Host "<+steps.read_pom.output.outputVariables.MAVEN_GROUP_ID>:<+steps.read_pom.output.outputVariables.MAVEN_ARTIFACT_ID>:<+steps.read_pom.output.outputVariables.MAVEN_VERSION>"
```

```yaml
# Effective version through Maven (original behavior)
- step:
    type: Plugin
    name: drone-get-maven-version-plugin
    identifier: maven_plugin
    spec:
      connectorRef: harness-docker-connector
      image: harnesscommunity/drone-get-maven-version:linux-amd64
      settings:
        pom_path: .

# Build and push the docker image with POM version as the tag
- step:
    type: BuildAndPushDockerRegistry
    name: BuildAndPushDockerRegistry
    identifier: BuildAndPushDockerRegistry
    spec:
      connectorRef: harness-docker-connector
      repo: namespace/container-name
      tags:
        - <+steps.maven_plugin.output.outputVariables.POM_VERSION>
```

## Development

```bash
go vet ./... && go test ./...                 # unit tests, no Maven needed
go test -tags integration ./...               # needs mvn on PATH and repository access
sh scripts/build.sh                           # binaries under release/
```

Release pipelines live in `.harness/` (same layout as `node-ci-images`):

| Pipeline | What it does |
| :-- | :-- |
| `validate.yaml` | `gofmt`, `go vet`, unit tests, binary builds, release gate (`scripts/check-release.sh`), secret scan. |
| `publish.yaml` | Per LTSC, on the `windows-2019` / `windows-2022` / `windows-2025` pool: checks the host build, refuses an existing tag, tests and builds `drone-maven.exe`, pushes only the immutable `windows-ltscXXXX-rN` tag, then runs `tests/contracts/Test-ImageContract.ps1` against the pushed image. |
| `qualify.yaml` | Per LTSC, on a `KubernetesDirect` Windows node with the matching build (`gcopdmwindowsbuildfarm` for 2019/2022, `gcopdmwindows2025` for 2025): runs the image contract, then runs the image as a Plugin step with only `pom_path` (backward compatibility) and as the Bamboo replacement (`pom_file`, `variable_prefix: maven`, `mode: raw_gav`, fixtures in `tests/qualify/`), and checks the step output variables. |
| `promote.yaml` | Moves `windows-ltscXXXX` (and, on request, the legacy `windows-amd64`) to a qualified digest with `crane`, without rebuilding. |

To release a changed image, bump `IMAGE_VERSION` in its `docker/Dockerfile.windows.amd64.ltscXXXX` and the tag in `.harness/publish.yaml` to the next `-rN`. The release gate fails if the two disagree, and publishing refuses to overwrite an existing `-rN`.

> <span style="font-size: 14px; margin-left:5px; background-color: #d3d3d3; padding: 4px; border-radius: 4px;">ℹ️ If you notice any issues in this documentation, you can [edit this document](https://github.com/harness-community/drone-get-maven-version/blob/main/README.md) to improve it.</span>
