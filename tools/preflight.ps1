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
    function Get-PackInfoDescription {
        param(
            [Parameter(Mandatory = $true)]
            [string]$Source
        )

        if ($Source -eq "WORKTREE") {
            $content = Get-Content -LiteralPath database_pack.info -Raw
        } else {
            $content = git show "${Source}:database_pack.info"
            if ($LASTEXITCODE -ne 0) {
                throw "Unable to read database_pack.info from git ref '$Source'."
            }
        }

        return ($content | ConvertFrom-Json).description
    }

    if (-not (Test-Path -LiteralPath $Database)) {
        throw "Database file not found: $Database"
    }

    if (-not $SkipDiffCheck) {
        $packInfoStatus = git status --porcelain -- database_pack.info
        if ($packInfoStatus) {
            $headDescription = Get-PackInfoDescription -Source "HEAD"
            $worktreeDescription = Get-PackInfoDescription -Source "WORKTREE"
            if ($worktreeDescription -ne $headDescription) {
                throw "database_pack.info description changed locally. Only non-description metadata fields, such as version, may change."
            }
            Write-Host "database_pack.info has local metadata changes; description field is unchanged."
        }

        if ($BaseRef) {
            git rev-parse --verify "$BaseRef^{commit}" *> $null
            if ($LASTEXITCODE -eq 0) {
                $range = "$BaseRef...HEAD"
                $changedFiles = git diff --name-only $range --
                if ($changedFiles -contains "database_pack.info") {
                    $baseDescription = Get-PackInfoDescription -Source $BaseRef
                    $headDescription = Get-PackInfoDescription -Source "HEAD"
                    if ($headDescription -ne $baseDescription) {
                        throw "database_pack.info description changed in $range. Only non-description metadata fields, such as version, may change."
                    }
                    Write-Host "database_pack.info changed in $range; description field is unchanged."
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
        -max-default-logo-refs 0 `
        -require-custom-logo-team "NRG" `
        -require-custom-logo-team "Deep Cross Gaming" `
        -require-custom-logo-team "VARREL YOUTH" `
        $Database

    if ($LASTEXITCODE -ne 0) {
        throw "TFM2DB validation failed."
    }
}
finally {
    Pop-Location
}