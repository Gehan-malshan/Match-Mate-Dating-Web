# Convenience entry point for developers working from services/event-service.
# The canonical cross-service smoke test remains under the repository root.
$canonicalSmokeTest = Join-Path $PSScriptRoot '..\..\..\..\tests\e2e\event-service-smoke.ps1'
$canonicalSmokeTest = (Resolve-Path -LiteralPath $canonicalSmokeTest).Path
& $canonicalSmokeTest @args
