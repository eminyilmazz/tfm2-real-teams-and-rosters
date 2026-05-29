[CmdletBinding()]
param(
    [string]$Database = "tfm2_teams_and_rosters.tfm2db",
    [string]$BaseRef = "",
    [switch]$SkipDiffCheck
)

$ErrorActionPreference = "Stop"

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
Push-Location $repoRoot
try {
    if (-not (Test-Path -LiteralPath $Database)) {
        throw "Database file not found: $Database"
    }

    if (-not $SkipDiffCheck) {
        $packInfoStatus = git status --porcelain -- database_pack.info
        if ($packInfoStatus) {
            throw "database_pack.info has local changes. Do not edit it; Steam behaves unpredictably when this file changes."
        }

        if ($BaseRef) {
            git rev-parse --verify "$BaseRef^{commit}" *> $null
            if ($LASTEXITCODE -eq 0) {
                $range = "$BaseRef...HEAD"
                $changedFiles = git diff --name-only $range --
                if ($changedFiles -contains "database_pack.info") {
                    throw "database_pack.info changed in $range. Revert it and keep package metadata stable."
                }
            } else {
                Write-Warning "Base ref '$BaseRef' is not available locally; skipped committed database_pack.info diff check."
            }
        }
    }

    $packInfoHash = (Get-FileHash -LiteralPath database_pack.info -Algorithm SHA256).Hash.ToLowerInvariant()
    $databaseHash = (Get-FileHash -LiteralPath $Database -Algorithm SHA256).Hash.ToLowerInvariant()
    Write-Host "database_pack.info sha256: $packInfoHash"
    Write-Host "$Database sha256: $databaseHash"

    & go run ./tools/validate.go `
        -strict `
        -expected-kind 1 `
        -expected-team-count 120 `
        -expected-custom-logo-blocks 120 `
        -max-default-logo-refs 2 `
        -allow-default-logo-team "Deep Cross Gaming" `
        -allow-default-logo-team "VARREL YOUTH" `
        -require-custom-logo-team "NRG" `
        $Database

    if ($LASTEXITCODE -ne 0) {
        throw "TFM2DB validation failed."
    }
}
finally {
    Pop-Location
}