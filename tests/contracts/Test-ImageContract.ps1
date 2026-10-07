[CmdletBinding()]
param(
    [Parameter(Mandatory)][ValidateSet('17763', '20348', '26100')][string]$ExpectedBuild,
    [string]$Plugin = 'C:\app\drone-maven.exe',
    # effective mode downloads maven-help-plugin from the configured
    # repositories; skip it where the qualification host has no access.
    [switch]$SkipEffective
)

# Runs inside the image under test, for example:
#   docker run --rm -v "${PWD}\tests:C:\tests" --entrypoint powershell <image> `
#     -NoProfile -File C:\tests\contracts\Test-ImageContract.ps1 -ExpectedBuild 20348

$ErrorActionPreference = 'Stop'
$failures = [System.Collections.Generic.List[string]]::new()

function Check([string]$Name, [scriptblock]$Test) {
    try {
        & $Test
        Write-Host "PASS  $Name"
    } catch {
        Write-Host "FAIL  $Name : $($_.Exception.Message)"
        $failures.Add($Name)
    }
}

# Windows PowerShell turns redirected native stderr into terminating errors
# under 'Stop'; java -version, for one, writes to stderr.
function Invoke-Native([string]$Exe, [string[]]$Arguments) {
    $ErrorActionPreference = 'Continue'
    $out = & $Exe @Arguments 2>&1 | Out-String
    if ($LASTEXITCODE -ne 0) { throw "$Exe $Arguments exited $LASTEXITCODE`n$out" }
    $out
}

# Runs the plugin with only the given PLUGIN_* settings and returns its exit
# code and DRONE_OUTPUT content.
function Invoke-Plugin([hashtable]$Settings) {
    $outFile = Join-Path $env:TEMP ("drone-output-{0}.env" -f [guid]::NewGuid())
    $names = @('PLUGIN_POM_PATH', 'PLUGIN_POM_FILE', 'PLUGIN_VARIABLE_PREFIX', 'PLUGIN_MODE')
    foreach ($n in $names) { Remove-Item "Env:$n" -ErrorAction SilentlyContinue }
    foreach ($k in $Settings.Keys) { Set-Item "Env:$k" $Settings[$k] }
    $env:DRONE_OUTPUT = $outFile
    $ErrorActionPreference = 'Continue'
    try {
        $log = & $Plugin 2>&1 | Out-String
        $code = $LASTEXITCODE
    } finally {
        foreach ($n in ($names + 'DRONE_OUTPUT')) { Remove-Item "Env:$n" -ErrorAction SilentlyContinue }
    }
    $content = if (Test-Path -LiteralPath $outFile) { [IO.File]::ReadAllText($outFile) } else { $null }
    Remove-Item -LiteralPath $outFile -ErrorAction SilentlyContinue
    [pscustomobject]@{ Code = $code; Output = $content; Log = $log }
}

$build = (Get-ItemProperty 'HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion').CurrentBuild
Check "Windows build is $ExpectedBuild" {
    if ($build -ne $ExpectedBuild) { throw "got $build" }
}

Check 'java -version' { Invoke-Native 'java.exe' @('-version') | Write-Host }
Check 'mvn --version' { Invoke-Native 'mvn.cmd' @('--version') | Write-Host }
Check 'git --version (inherited from ci-base)' { Invoke-Native 'git.exe' @('--version') | Write-Host }

Check 'JAVA_HOME and MAVEN_HOME' {
    if ($env:JAVA_HOME -ne 'C:\tools\java') { throw "JAVA_HOME=$env:JAVA_HOME" }
    if ($env:MAVEN_HOME -ne 'C:\tools\maven') { throw "MAVEN_HOME=$env:MAVEN_HOME" }
    if (-not (Test-Path "$env:JAVA_HOME\bin\java.exe")) { throw 'java.exe missing under JAVA_HOME' }
    if (-not (Test-Path "$env:MAVEN_HOME\bin\mvn.cmd")) { throw 'mvn.cmd missing under MAVEN_HOME' }
}

Check 'PATH extends the ci-base PATH' {
    $entries = $env:PATH -split ';'
    foreach ($want in 'C:\tools\maven\bin', 'C:\tools\java\bin', 'C:\tools\git\cmd', 'C:\Windows\System32', 'C:\Windows') {
        if ($entries -notcontains $want) { throw "PATH is missing $want" }
    }
}

# Fixtures live in a workspace whose path contains spaces.
$workspace = Join-Path $env:TEMP 'contract work space'
$project = Join-Path $workspace 'my app'
$child = Join-Path $project 'child module'
New-Item -ItemType Directory -Force -Path $child | Out-Null
Set-Content -LiteralPath (Join-Path $project 'pom.xml') -Encoding UTF8 -Value @'
<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0">
  <modelVersion>4.0.0</modelVersion>
  <groupId>com.example.contract</groupId>
  <artifactId>contract-parent</artifactId>
  <version>1.2.3-SNAPSHOT</version>
  <packaging>pom</packaging>
</project>
'@
Set-Content -LiteralPath (Join-Path $child 'pom.xml') -Encoding UTF8 -Value @'
<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0">
  <modelVersion>4.0.0</modelVersion>
  <parent>
    <groupId>com.example.contract</groupId>
    <artifactId>contract-parent</artifactId>
    <version>1.2.3-SNAPSHOT</version>
    <relativePath>../pom.xml</relativePath>
  </parent>
  <artifactId>contract-child</artifactId>
  <dependencies>
    <dependency>
      <groupId>leak.example</groupId>
      <artifactId>leak</artifactId>
      <version>9.9.9</version>
    </dependency>
  </dependencies>
</project>
'@

Push-Location -LiteralPath $workspace
try {
    Check 'raw_gav with pom_file, prefix maven, relative path with spaces' {
        $r = Invoke-Plugin @{
            PLUGIN_POM_FILE        = 'my app\child module\pom.xml'
            PLUGIN_VARIABLE_PREFIX = 'maven'
            PLUGIN_MODE            = 'raw_gav'
        }
        if ($r.Code -ne 0) { throw "exit $($r.Code)`n$($r.Log)" }
        $want = "MAVEN_GROUP_ID=com.example.contract`nMAVEN_ARTIFACT_ID=contract-child`nMAVEN_VERSION=1.2.3-SNAPSHOT`nPOM_VERSION=1.2.3-SNAPSHOT`n"
        if ($r.Output -cne $want) { throw "DRONE_OUTPUT was:`n$($r.Output)" }
    }

    Check 'raw_gav with an absolute forward-slash path' {
        $r = Invoke-Plugin @{
            PLUGIN_POM_FILE = ((Join-Path $child 'pom.xml') -replace '\\', '/')
            PLUGIN_MODE     = 'raw_gav'
        }
        if ($r.Code -ne 0) { throw "exit $($r.Code)`n$($r.Log)" }
        if ($r.Output -notmatch '(?m)^MAVEN_ARTIFACT_ID=contract-child$') { throw "DRONE_OUTPUT was:`n$($r.Output)" }
    }

    Check 'missing POM fails without writing outputs' {
        $r = Invoke-Plugin @{ PLUGIN_POM_FILE = 'does not exist\pom.xml'; PLUGIN_MODE = 'raw_gav' }
        if ($r.Code -eq 0) { throw 'expected a non-zero exit' }
        if ($r.Output) { throw "unexpected DRONE_OUTPUT: $($r.Output)" }
    }

    Check 'no settings fails like the old plugin' {
        $r = Invoke-Plugin @{}
        if ($r.Code -ne 1) { throw "exit $($r.Code)" }
        if ($r.Log -notmatch 'POM Path is empty') { throw $r.Log }
    }

    if ($SkipEffective) {
        Write-Host 'SKIP  effective mode (-SkipEffective)'
    } else {
        Check 'effective mode with only pom_path (backward compatible)' {
            $r = Invoke-Plugin @{ PLUGIN_POM_PATH = 'my app\child module' }
            if ($r.Code -ne 0) { throw "exit $($r.Code)`n$($r.Log)" }
            if ($r.Output -cne "POM_VERSION=1.2.3-SNAPSHOT`n") { throw "DRONE_OUTPUT was:`n$($r.Output)" }
        }
    }
} finally {
    Pop-Location
    Remove-Item -LiteralPath $workspace -Recurse -Force -ErrorAction SilentlyContinue
}

Check 'no leftover installers or caches' {
    $leftovers = @(
        'C:\ProgramData\chocolatey\lib-bad',
        'C:\ProgramData\chocolatey\logs',
        (Join-Path $env:TEMP 'chocolatey'),
        'C:\app\.git'
    ) | Where-Object { Test-Path -LiteralPath $_ }
    $installers = @(Get-ChildItem -Path $env:TEMP, C:\Windows\Temp, C:\app -Recurse -File -Force `
        -Include *.msi, *.nupkg, *.zip -ErrorAction SilentlyContinue)
    $all = @($leftovers) + @($installers | ForEach-Object FullName)
    if ($all.Count -gt 0) { throw ($all -join ', ') }
}

if ($failures.Count -gt 0) {
    Write-Host "`n$($failures.Count) contract check(s) failed: $($failures -join '; ')"
    exit 1
}
Write-Host "`nAll contract checks passed on Windows build $build."
