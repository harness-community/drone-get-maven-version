[CmdletBinding()]
param(
    [Parameter(Mandatory)][ValidateSet('ltsc2019', 'ltsc2022', 'ltsc2025')][string]$Ltsc,
    [Parameter(Mandatory)][ValidateSet('host', 'binary', 'tag-free')][string]$Step,
    [string]$Repository = 'harnesscommunity/drone-get-maven-version'
)

# Helper steps for the Windows publish stages in .harness/publish.yaml. They
# run on the VM host of the pool whose Windows build matches the LTSC.
#   host      fail unless the host Windows build matches the LTSC
#   binary    test and build release/windows/amd64/drone-maven.exe with a pinned Go
#             (COMMIT_SHA and BUILD_ID come from the pipeline step)
#   tag-free  fail if the immutable release tag already exists on Docker Hub

$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'

$expectedBuild = @{ ltsc2019 = '17763'; ltsc2022 = '20348'; ltsc2025 = '26100' }[$Ltsc]
$dockerfile = "docker/Dockerfile.windows.amd64.$Ltsc"
$goVersion = '1.25.12'
$goSha256 = 'd5dc82da351b00e5eedd04f41356817d674cc4308131f0f638a5b14c5c3af4cb'

function Invoke-Checked([string]$Exe, [string[]]$Arguments) {
    & $Exe @Arguments
    if ($LASTEXITCODE -ne 0) { throw "$Exe $($Arguments -join ' ') exited $LASTEXITCODE" }
}

switch ($Step) {
    'host' {
        $build = (Get-ItemProperty 'HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion').CurrentBuild
        if ($build -ne $expectedBuild) {
            throw "host Windows build is $build; $Ltsc images need build $expectedBuild for process isolation"
        }
        Write-Host "host Windows build $build matches $Ltsc"
    }
    'binary' {
        # golang:<ver> has no Windows Server 2019 image, so use the official
        # zip, verified against its published checksum, on every LTSC.
        $root = Join-Path $env:TEMP "go$goVersion"
        $zip = "$root.zip"
        if (-not (Test-Path "$root\go\bin\go.exe")) {
            Invoke-WebRequest -UseBasicParsing -Uri "https://go.dev/dl/go$goVersion.windows-amd64.zip" -OutFile $zip
            $sha = (Get-FileHash -Algorithm SHA256 -LiteralPath $zip).Hash.ToLowerInvariant()
            if ($sha -ne $goSha256) { throw "go$goVersion.windows-amd64.zip sha256 is $sha, want $goSha256" }
            Expand-Archive -LiteralPath $zip -DestinationPath $root -Force
            Remove-Item -LiteralPath $zip
        }
        $env:CGO_ENABLED = '0'
        $env:GOOS = 'windows'
        $env:GOARCH = 'amd64'
        $commit = "$env:COMMIT_SHA"
        $version = if ($commit.Length -ge 8) { $commit.Substring(0, 8) } else { 'dev' }
        $ldflags = "-X main.version=$version -X main.build=$env:BUILD_ID"
        Invoke-Checked "$root\go\bin\go.exe" @('test', './...')
        Invoke-Checked "$root\go\bin\go.exe" @('build', '-ldflags', $ldflags, '-o', 'release/windows/amd64/drone-maven.exe', '.')
    }
    'tag-free' {
        $m = Select-String -LiteralPath $dockerfile -Pattern '^ARG IMAGE_VERSION=(\S+)\s*$'
        if (-not $m) { throw "$dockerfile has no ARG IMAGE_VERSION" }
        $tag = $m.Matches[0].Groups[1].Value
        if ($tag -notmatch "^windows-$Ltsc-r[1-9][0-9]*$") { throw "IMAGE_VERSION $tag does not match windows-$Ltsc-rN" }
        $uri = "https://hub.docker.com/v2/repositories/$Repository/tags/$tag"
        try {
            Invoke-WebRequest -UseBasicParsing -Uri $uri | Out-Null
            throw "${Repository}:$tag already exists and release tags are immutable; bump IMAGE_VERSION in $dockerfile and the tag in .harness/publish.yaml"
        } catch [System.Net.WebException] {
            if ($_.Exception.Response -and [int]$_.Exception.Response.StatusCode -eq 404) {
                Write-Host "${Repository}:$tag is free"
            } else { throw }
        }
    }
}
