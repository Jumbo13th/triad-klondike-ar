# Regenerates the per-language StringTableRuntime confs from tk_localization.st.
# A language without a Target_<lang> entry gets the en_us text, which is what
# Workbench's own export does; the runtime confs have no engine-side fallback.
#
# Usage: pwsh -File tools/generate-runtime-locales.ps1
# Requires every Target_* value in the .st on one line with no embedded quotes.

$ErrorActionPreference = 'Stop'

$langDir = Join-Path $PSScriptRoot '..\Triad Klondike\Language'
$stPath  = Join-Path $langDir 'tk_localization.st'
$languages = @('cs_cz','de_de','en_us','es_es','fr_fr','it_it','ja_jp','ko_kr','pl_pl','pt_br','ru_ru','uk_ua','zh_cn')

$items = @()
$current = $null
foreach ($line in Get-Content -Path $stPath -Encoding UTF8) {
    if ($line -match '^\s*Id\s+"(.+)"\s*$') {
        $current = @{ Id = $Matches[1]; Targets = @{} }
        $items += $current
    }
    elseif ($line -match '^\s*Target_([a-z]{2}_[a-z]{2})\s+"(.*)"\s*$') {
        if ($null -eq $current) { throw "Target line before any Id line: $line" }
        $current.Targets[$Matches[1]] = $Matches[2]
    }
}

if ($items.Count -eq 0) { throw "No items parsed from $stPath" }
foreach ($item in $items) {
    if (-not $item.Targets.ContainsKey('en_us')) { throw "Item '$($item.Id)' has no Target_en_us" }
}

$utf8NoBom = New-Object System.Text.UTF8Encoding($false)

foreach ($lang in $languages) {
    $sb = New-Object System.Text.StringBuilder
    [void]$sb.AppendLine('StringTableRuntime {')
    [void]$sb.AppendLine(' Ids {')
    foreach ($item in $items) { [void]$sb.AppendLine('  "' + $item.Id + '"') }
    [void]$sb.AppendLine(' }')
    [void]$sb.AppendLine(' Texts {')
    foreach ($item in $items) {
        $text = $item.Targets[$lang]
        if ([string]::IsNullOrEmpty($text)) { $text = $item.Targets['en_us'] }
        [void]$sb.AppendLine('  "' + $text + '"')
    }
    [void]$sb.AppendLine(' }')
    [void]$sb.AppendLine('}')

    $outPath = [System.IO.Path]::GetFullPath((Join-Path $langDir "tk_localization.$lang.conf"))
    [System.IO.File]::WriteAllText($outPath, $sb.ToString(), $utf8NoBom)
}

Write-Host "Wrote $($languages.Count) runtime confs, $($items.Count) keys each."
