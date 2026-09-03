#requires -Version 7.0
<#
.SYNOPSIS
Renders LORE-DOSSIER-EN.pdf and LORE-DOSSIER-RU.pdf from the print HTML in this folder.

.DESCRIPTION
print.css owns the page chrome and layout-check.js the runtime layout check,
which runs first: every page must report no overflow and the
table of contents must resolve. Chromium then prints each edition straight to docs/.
#>
[CmdletBinding()]
param(
    [string]$Browser
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$LoreDirectory = Split-Path -Parent $PSCommandPath
$DocsDirectory = Split-Path -Parent $LoreDirectory
$Editions = @(
    @{ Lang = 'en'; Html = Join-Path $LoreDirectory 'lore-dossier-en.html'; Pdf = Join-Path $DocsDirectory 'LORE-DOSSIER-EN.pdf' },
    @{ Lang = 'ru'; Html = Join-Path $LoreDirectory 'lore-dossier-ru.html'; Pdf = Join-Path $DocsDirectory 'LORE-DOSSIER-RU.pdf' }
)

function Resolve-Browser {
    param([string]$Requested)

    if ($Requested) {
        if (-not (Test-Path -LiteralPath $Requested -PathType Leaf)) {
            throw "Chromium browser not found: $Requested"
        }
        return (Resolve-Path -LiteralPath $Requested).Path
    }

    $candidates = @(
        'C:\Program Files\Google\Chrome\Application\chrome.exe',
        'C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe',
        'C:\Program Files\Microsoft\Edge\Application\msedge.exe'
    )
    foreach ($candidate in $candidates) {
        if (Test-Path -LiteralPath $candidate -PathType Leaf) {
            return $candidate
        }
    }
    throw 'Chrome or Edge is required to render the lore PDFs.'
}

$commonArguments = @(
    '--headless',
    '--disable-gpu',
    '--disable-background-mode',
    '--disable-background-networking',
    '--disable-component-update',
    '--disable-default-apps',
    '--disable-extensions',
    '--disable-sync',
    '--metrics-recording-only',
    '--no-first-run',
    '--allow-file-access-from-files'
)

function Assert-RuntimeLayout {
    param([string]$BrowserPath, [string]$Profile, [string]$HtmlPath)

    $uri = ([System.Uri]$HtmlPath).AbsoluteUri
    $arguments = $commonArguments + @('--virtual-time-budget=2000', "--user-data-dir=$Profile", '--dump-dom', $uri)
    $dom = (& $BrowserPath @arguments 2>$null | Out-String)
    if ($dom -notmatch 'data-layout-ready="true"') {
        throw "Layout validation did not finish for $HtmlPath."
    }
    if ($dom -notmatch 'data-layout-errors=""') {
        $match = [regex]::Match($dom, 'data-layout-errors="([^"]*)"')
        $detail = if ($match.Success) { $match.Groups[1].Value } else { 'layout check did not report' }
        throw "Layout validation failed for ${HtmlPath}: $detail"
    }
    $pages = [regex]::Match($dom, 'data-rendered-pages="(\d+)"').Groups[1].Value
    return [int]$pages
}

function New-Pdf {
    param([string]$BrowserPath, [string]$Profile, [string]$HtmlPath, [string]$PdfPath)

    if (Test-Path -LiteralPath $PdfPath) {
        Remove-Item -LiteralPath $PdfPath -Force
    }
    $uri = ([System.Uri]$HtmlPath).AbsoluteUri
    $arguments = $commonArguments + @(
        '--no-pdf-header-footer',
        '--generate-pdf-document-outline',
        "--user-data-dir=$Profile",
        "--print-to-pdf=$PdfPath",
        $uri
    )
    & $BrowserPath @arguments 2>$null | Out-Null
    foreach ($attempt in 1..50) {
        if (Test-Path -LiteralPath $PdfPath -PathType Leaf) { break }
        Start-Sleep -Milliseconds 100
    }
    if (-not (Test-Path -LiteralPath $PdfPath -PathType Leaf)) {
        throw "PDF rendering did not create $PdfPath."
    }
}

$browserPath = Resolve-Browser -Requested $Browser
$profile = Join-Path ([System.IO.Path]::GetTempPath()) ("lore-dossier-chromium-" + [guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $profile -Force | Out-Null

try {
    foreach ($edition in $Editions) {
        $pages = Assert-RuntimeLayout -BrowserPath $browserPath -Profile $profile -HtmlPath $edition.Html
        New-Pdf -BrowserPath $browserPath -Profile $profile -HtmlPath $edition.Html -PdfPath $edition.Pdf
        $size = (Get-Item -LiteralPath $edition.Pdf).Length
        Write-Host ("Rendered {0}: {1} pages, {2:N0} bytes -> {3}" -f $edition.Lang.ToUpperInvariant(), $pages, $size, $edition.Pdf)
    }
}
finally {
    try { Remove-Item -LiteralPath $profile -Recurse -Force -ErrorAction Stop } catch { }
}
