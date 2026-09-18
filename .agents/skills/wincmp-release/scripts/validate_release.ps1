$ScriptDir = $PSScriptRoot
$JsPath = Join-Path $ScriptDir "validate_release.js"
node $JsPath
exit $LASTEXITCODE
