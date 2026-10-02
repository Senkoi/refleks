param(
    [Parameter(Mandatory = $true)][string]$GameRoot,
    [string]$NamesFile = (Join-Path $PSScriptRoot '..\first-family-names.txt'),
    [string]$OutputZip = (Join-Path (Get-Location) 'sce-first-family.zip'),
    [string]$DownloadNote = 'freshness_not_reported'
)
$ErrorActionPreference = 'Stop'
if (Test-Path -LiteralPath $OutputZip) { throw "Output already exists: $OutputZip" }
$wanted = [System.Collections.Generic.HashSet[string]]::new([System.StringComparer]::Ordinal)
foreach ($name in [IO.File]::ReadAllLines((Resolve-Path -LiteralPath $NamesFile).Path)) {
    if ($name.Length -gt 0) { [void]$wanted.Add($name) }
}
$resolvedRoot = (Resolve-Path -LiteralPath $GameRoot).Path
$roots = @(@(
    (Join-Path $resolvedRoot 'FPSAimTrainer\Saved\SaveGames\Scenarios'),
    (Join-Path $resolvedRoot 'Saved\SaveGames\Scenarios')
) | Where-Object { Test-Path -LiteralPath $_ -PathType Container })
# Workshop files can reside beside the Steam common directory.
$commonDir = Split-Path $resolvedRoot -Parent
if ((Split-Path $commonDir -Leaf) -eq 'common') {
    $steamappsDir = Split-Path $commonDir -Parent
    $workshopRoot = Join-Path $steamappsDir 'workshop\content\824270'
    if (Test-Path -LiteralPath $workshopRoot -PathType Container) { $roots += $workshopRoot }
}
if ($roots.Count -eq 0) { throw 'No scenario directory found under the supplied game root.' }
$stagePath = Join-Path ([IO.Path]::GetTempPath()) ('refleks-sce-' + [Guid]::NewGuid().ToString('N'))
[void](New-Item -ItemType Directory -Path $stagePath)
$records = [System.Collections.Generic.List[object]]::new()
$found = [System.Collections.Generic.HashSet[string]]::new([System.StringComparer]::Ordinal)
try {
    foreach ($root in $roots) {
        foreach ($file in (Get-ChildItem -LiteralPath $root -Filter '*.sce' -Recurse -File)) {
            if ($file.Length -gt 20000000) { continue }
            $reader = [IO.StreamReader]::new($file.FullName, [Text.Encoding]::UTF8, $true)
            $internalName = $null
            $gameVersion = $null
            try {
                while ($null -ne ($line = $reader.ReadLine())) {
                    if ($line.TrimStart().StartsWith('[')) { break }
                    if ($line -match '^Name=(.*)$') { $internalName = $Matches[1].Trim() }
                    if ($line -match '^GameVersion=(.*)$') { $gameVersion = $Matches[1].Trim() }
                }
            } finally { $reader.Dispose() }
            if (-not $wanted.Contains($internalName)) { continue }
            $hash = (Get-FileHash -LiteralPath $file.FullName -Algorithm SHA256).Hash.ToLowerInvariant()
            $destinationName = $hash + '.sce'
            $destination = Join-Path $stagePath $destinationName
            if (-not (Test-Path -LiteralPath $destination)) {
                Copy-Item -LiteralPath $file.FullName -Destination $destination
                $copyHash = (Get-FileHash -LiteralPath $destination -Algorithm SHA256).Hash.ToLowerInvariant()
                if ($copyHash -ne $hash) { throw 'Scenario changed while copying. Retry after downloads finish.' }
            }
            [void]$found.Add($internalName)
            $records.Add([PSCustomObject]@{
                scenario_name = $internalName
                original_filename = $file.Name
                archive_filename = $destinationName
                sha256 = $hash
                game_version_raw = $gameVersion
                byte_count = $file.Length
                file_mtime_utc = $file.LastWriteTimeUtc.ToString('o')
                current_version_verified = $false
            })
        }
    }
    $missing = @($wanted | Where-Object { -not $found.Contains($_) } | Sort-Object)
    $manifest = [PSCustomObject]@{
        collected_at = [DateTime]::UtcNow.ToString('o')
        download_note = $DownloadNote
        files = @($records.ToArray())
        missing_names = $missing
        calibration_eligible = $false
    }
    [IO.File]::WriteAllText((Join-Path $stagePath 'manifest.json'), ($manifest | ConvertTo-Json -Depth 8), [Text.UTF8Encoding]::new($false))
    Compress-Archive -Path (Join-Path $stagePath '*') -DestinationPath $OutputZip
    Write-Host ('Collected ' + $found.Count + '/' + $wanted.Count + ' scenarios: ' + $OutputZip)
    if ($missing.Count -gt 0) { Write-Host ('Missing: ' + ($missing -join ', ')) }
} finally {
    Remove-Item -LiteralPath $stagePath -Recurse -Force
}
