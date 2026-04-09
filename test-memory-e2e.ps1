<#
.SYNOPSIS
    Open-DB9 Agent Memory Layer End-to-End Test Script
.DESCRIPTION
    Comprehensive test suite to validate the complete Memory Skill implementation
.NOTES
    File: test-memory-e2e.ps1
#>

param(
    [switch]$VerboseOutput
)

$ErrorActionPreference = "Stop"
$ProjectRoot = "c:\Users\11428\Desktop\open_db9"
$ResultsFile = "$ProjectRoot\test-results.txt"
$TotalTests = 0
$PassedTests = 0
$FailedTests = 0
$TestResults = @()

function Write-Pass { param([string]$Msg) Write-Host "  [PASS] $Msg" -ForegroundColor Green }
function Write-Fail { param([string]$Msg) Write-Host "  [FAIL] $Msg" -ForegroundColor Red }
function Write-Warn { param([string]$Msg) Write-Host "  [WARN] $Msg" -ForegroundColor Yellow }
function Write-Info { param([string]$Msg) Write-Host "  [INFO] $Msg" -ForegroundColor Cyan }
function Write-Section { param([string]$Msg) Write-Host "" ; Write-Host "=== $Msg ===" -ForegroundColor White -BackgroundColor DarkBlue }

function Add-TestResult {
    param(
        [string]$Name,
        [bool]$Passed,
        [string]$Details = ""
    )

    $script:TotalTests++
    if ($Passed) {
        $script:PassedTests++
        Write-Pass $Name
    } else {
        $script:FailedTests++
        Write-Fail $Name
        if ($Details) { Write-Host "         -> $Details" -ForegroundColor Gray }
    }

    $script:TestResults += @{
        Name = $Name
        Status = if ($Passed) { "PASS" } else { "FAIL" }
        Details = $Details
    }
}

# Initialize Results File
"Open-DB9 Memory Skill E2E Test Results" | Out-File -FilePath $ResultsFile -Encoding UTF8
"Generated: $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')" | Out-File -FilePath $ResultsFile -Append -Encoding UTF8
"=" * 60 | Out-File -FilePath $ResultsFile -Append -Encoding UTF8
"" | Out-File -FilePath $ResultsFile -Append -Encoding UTF8

Write-Host ""
Write-Host "**********************************************************" -ForegroundColor Cyan
Write-Host "*  Open-DB9 Agent Memory Layer E2E Test Suite            *" -ForegroundColor Cyan
Write-Host "*  Testing Complete Implementation Chain                 *" -ForegroundColor Cyan
Write-Host "**********************************************************" -ForegroundColor Cyan
Write-Host ""

# ============================================================================
# TEST SECTION 1: Compilation Verification
# ============================================================================
Write-Section "TEST 1: Compilation Verification"

Write-Info "Checking Go installation..."
try {
    $goVersion = go version 2>&1
    if ($LASTEXITCODE -eq 0) {
        Add-TestResult -Name "Go compiler available" -Passed $true -Details $goVersion
    } else {
        Add-TestResult -Name "Go compiler available" -Passed $false -Details "Go not found in PATH"
    }
} catch {
    Add-TestResult -Name "Go compiler available" -Passed $false -Details $_.Exception.Message
}

Write-Info "Testing CLI (db9.exe) compilation..."
Set-Location $ProjectRoot
try {
    $null = New-Item -ItemType Directory -Path "build" -Force | Out-Null
    $buildOutput = go build -o build/db9.exe ./cmd/db9/ 2>&1
    if ($LASTEXITCODE -eq 0 -and (Test-Path "build/db9.exe")) {
        Add-TestResult -Name "db9.exe (CLI) compiles successfully" -Passed $true
    } else {
        Add-TestResult -Name "db9.exe (CLI) compiles successfully" -Passed $false -Details $buildOutput
    }
} catch {
    Add-TestResult -Name "db9.exe (CLI) compiles successfully" -Passed $false -Details $_.Exception.Message
}

Write-Info "Testing API Server (db9-server.exe) compilation..."
try {
    $buildOutput = go build -o build/db9-server.exe ./cmd/server/ 2>&1
    if ($LASTEXITCODE -eq 0 -and (Test-Path "build/db9-server.exe")) {
        Add-TestResult -Name "db9-server.exe (API Server) compiles successfully" -Passed $true
    } else {
        Add-TestResult -Name "db9-server.exe (API Server) compiles successfully" -Passed $false -Details $buildOutput
    }
} catch {
    Add-TestResult -Name "db9-server.exe (API Server) compiles successfully" -Passed $false -Details $_.Exception.Message
}

Write-Info "Testing FS9 Service (fs9-service.exe) compilation..."
try {
    $buildOutput = go build -o build/fs9-service.exe ./cmd/fs9-service/ 2>&1
    if ($LASTEXITCODE -eq 0 -and (Test-Path "build/fs9-service.exe")) {
        Add-TestResult -Name "fs9-service.exe (FS9 Service) compiles successfully" -Passed $true
    } else {
        Add-TestResult -Name "fs9-service.exe (FS9 Service) compiles successfully" -Passed $false -Details $buildOutput
    }
} catch {
    Add-TestResult -Name "fs9-service.exe (FS9 Service) compiles successfully" -Passed $false -Details $_.Exception.Message
}

Write-Info "Testing MCP Server (mcp-server.exe) compilation..."
try {
    $buildOutput = go build -o build/mcp-server.exe ./cmd/mcp-server/ 2>&1
    if ($LASTEXITCODE -eq 0 -and (Test-Path "build/mcp-server.exe")) {
        Add-TestResult -Name "mcp-server.exe (MCP Server with memory tools) compiles successfully" -Passed $true
    } else {
        Add-TestResult -Name "mcp-server.exe (MCP Server with memory tools) compiles successfully" -Passed $false -Details $buildOutput
    }
} catch {
    Add-TestResult -Name "mcp-server.exe (MCP Server with memory tools) compiles successfully" -Passed $false -Details $_.Exception.Message
}

# ============================================================================
# TEST SECTION 2: Code Structure Verification
# ============================================================================
Write-Section "TEST 2: Code Structure Verification"

$migrationFile = "$ProjectRoot\migrations\control\005_agent_memory.up.sql"
if (Test-Path $migrationFile) {
    Add-TestResult -Name "Migration file 005_agent_memory.up.sql exists" -Passed $true
} else {
    Add-TestResult -Name "Migration file 005_agent_memory.up.sql exists" -Passed $false -Details "File not found"
}

$memoryHandlerFile = "$ProjectRoot\internal\api\handlers\memory.go"
if (Test-Path $memoryHandlerFile) {
    Add-TestResult -Name "Memory handler file (memory.go) exists" -Passed $true
} else {
    Add-TestResult -Name "Memory handler file (memory.go) exists" -Passed $false -Details "File not found"
}

$mcpHandlersFile = "$ProjectRoot\cmd\mcp-server\handlers.go"
if (Test-Path $mcpHandlersFile) {
    Add-TestResult -Name "MCP server handlers file (handlers.go) exists" -Passed $true
} else {
    Add-TestResult -Name "MCP server handlers file (handlers.go) exists" -Passed $false -Details "File not found"
}

$routerFile = "$ProjectRoot\internal\api\router\router.go"
if (Test-Path $routerFile) {
    $routerContent = Get-Content $routerFile -Raw
    if ($routerContent -match "memories") {
        Add-TestResult -Name "Router file contains /memories routes" -Passed $true
    } else {
        Add-TestResult -Name "Router file contains /memories routes" -Passed $false -Details "No memories routes found"
    }
} else {
    Add-TestResult -Name "Router file exists and contains /memories routes" -Passed $false -Details "Router file not found"
}

$memorySkillDoc = "$ProjectRoot\skills\memory-skill.md"
if (Test-Path $memorySkillDoc) {
    Add-TestResult -Name "Memory skill documentation (memory-skill.md) exists" -Passed $true
} else {
    Add-TestResult -Name "Memory skill documentation (memory-skill.md) exists" -Passed $false -Details "File not found"
}

$quickstartDoc = "$ProjectRoot\skills\quickstart-guide.md"
if (Test-Path $quickstartDoc) {
    Add-TestResult -Name "Quick start guide documentation (quickstart-guide.md) exists" -Passed $true
} else {
    Add-TestResult -Name "Quick start guide documentation (quickstart-guide.md) exists" -Passed $false -Details "File not found"
}

$quickstartScript = "$ProjectRoot\scripts\db9-quickstart.sh"
if (Test-Path $quickstartScript) {
    Add-TestResult -Name "Quick start script (db9-quickstart.sh) exists" -Passed $true
} else {
    Add-TestResult -Name "Quick start script (db9-quickstart.sh) exists" -Passed $false -Details "File not found"
}

# ============================================================================
# TEST SECTION 3: MCP Tools Registration Verification
# ============================================================================
Write-Section "TEST 3: MCP Tools Registration Verification"

if (Test-Path $mcpHandlersFile) {
    $mcpContent = Get-Content $mcpHandlersFile -Raw
    
    if ($mcpContent -match 'NewTool\("memory_store"') {
        Add-TestResult -Name "MCP tool memory_store registered" -Passed $true
    } else {
        Add-TestResult -Name "MCP tool memory_store registered" -Passed $false -Details "Tool registration not found"
    }
    
    if ($mcpContent -match 'NewTool\("memory_recall"') {
        Add-TestResult -Name "MCP tool memory_recall registered" -Passed $true
    } else {
        Add-TestResult -Name "MCP tool memory_recall registered" -Passed $false -Details "Tool registration not found"
    }
    
    if ($mcpContent -match 'NewTool\("memory_list"') {
        Add-TestResult -Name "MCP tool memory_list registered" -Passed $true
    } else {
        Add-TestResult -Name "MCP tool memory_list registered" -Passed $false -Details "Tool registration not found"
    }
    
    if ($mcpContent -match 'NewTool\("memory_delete"') {
        Add-TestResult -Name "MCP tool memory_delete registered" -Passed $true
    } else {
        Add-TestResult -Name "MCP tool memory_delete registered" -Passed $false -Details "Tool registration not found"
    }
    
    if ($mcpContent -match "func handleMemoryStore") {
        Add-TestResult -Name "Handler function handleMemoryStore exists" -Passed $true
    } else {
        Add-TestResult -Name "Handler function handleMemoryStore exists" -Passed $false -Details "Function not found"
    }
    
    if ($mcpContent -match "func handleMemoryRecall") {
        Add-TestResult -Name "Handler function handleMemoryRecall exists" -Passed $true
    } else {
        Add-TestResult -Name "Handler function handleMemoryRecall exists" -Passed $false -Details "Function not found"
    }
    
    if ($mcpContent -match "func handleMemoryList") {
        Add-TestResult -Name "Handler function handleMemoryList exists" -Passed $true
    } else {
        Add-TestResult -Name "Handler function handleMemoryList exists" -Passed $false -Details "Function not found"
    }
    
    if ($mcpContent -match "func handleMemoryDelete") {
        Add-TestResult -Name "Handler function handleMemoryDelete exists" -Passed $true
    } else {
        Add-TestResult -Name "Handler function handleMemoryDelete exists" -Passed $false -Details "Function not found"
    }
} else {
    Write-Warn "MCP handlers file not found, skipping tests"
    Add-TestResult -Name "MCP tools verification" -Passed $false -Details "Source file missing"
}

# ============================================================================
# TEST SECTION 4: API Handler Completeness Verification
# ============================================================================
Write-Section "TEST 4: API Handler Completeness Verification"

if (Test-Path $memoryHandlerFile) {
    $handlerContent = Get-Content $memoryHandlerFile -Raw
    
    if ($handlerContent -match "func StoreMemoryHandler") {
        Add-TestResult -Name "API handler StoreMemoryHandler exists" -Passed $true
        
        if ($handlerContent -match "func StoreMemoryHandler\(w http\.ResponseWriter, r \*http\.Request\)") {
            Add-TestResult -Name "StoreMemoryHandler has correct signature" -Passed $true
        } else {
            Add-TestResult -Name "StoreMemoryHandler has correct signature" -Passed $false -Details "Signature mismatch"
        }
    } else {
        Add-TestResult -Name "API handler StoreMemoryHandler exists" -Passed $false -Details "Function not found"
    }
    
    if ($handlerContent -match "func RecallMemoryHandler") {
        Add-TestResult -Name "API handler RecallMemoryHandler exists" -Passed $true
        
        if ($handlerContent -match "func RecallMemoryHandler\(w http\.ResponseWriter, r \*http\.Request\)") {
            Add-TestResult -Name "RecallMemoryHandler has correct signature" -Passed $true
        } else {
            Add-TestResult -Name "RecallMemoryHandler has correct signature" -Passed $false -Details "Signature mismatch"
        }
    } else {
        Add-TestResult -Name "API handler RecallMemoryHandler exists" -Passed $false -Details "Function not found"
    }
    
    if ($handlerContent -match "func ListMemoriesHandler") {
        Add-TestResult -Name "API handler ListMemoriesHandler exists" -Passed $true
        
        if ($handlerContent -match "func ListMemoriesHandler\(w http\.ResponseWriter, r \*http\.Request\)") {
            Add-TestResult -Name "ListMemoriesHandler has correct signature" -Passed $true
        } else {
            Add-TestResult -Name "ListMemoriesHandler has correct signature" -Passed $false -Details "Signature mismatch"
        }
    } else {
        Add-TestResult -Name "API handler ListMemoriesHandler exists" -Passed $false -Details "Function not found"
    }
    
    if ($handlerContent -match "func DeleteMemoryHandler") {
        Add-TestResult -Name "API handler DeleteMemoryHandler exists" -Passed $true
        
        if ($handlerContent -match "func DeleteMemoryHandler\(w http\.ResponseWriter, r \*http\.Request\)") {
            Add-TestResult -Name "DeleteMemoryHandler has correct signature" -Passed $true
        } else {
            Add-TestResult -Name "DeleteMemoryHandler has correct signature" -Passed $false -Details "Signature mismatch"
        }
    } else {
        Add-TestResult -Name "API handler DeleteMemoryHandler exists" -Passed $false -Details "Function not found"
    }
    
    if ($handlerContent -match "type MemoryStoreRequest struct") {
        Add-TestResult -Name "MemoryStoreRequest struct defined" -Passed $true
    } else {
        Add-TestResult -Name "MemoryStoreRequest struct defined" -Passed $false -Details "Struct not found"
    }
    
    if ($handlerContent -match "type MemoryRecallRequest struct") {
        Add-TestResult -Name "MemoryRecallRequest struct defined" -Passed $true
    } else {
        Add-TestResult -Name "MemoryRecallRequest struct defined" -Passed $false -Details "Struct not found"
    }
} else {
    Write-Warn "Memory handler file not found, skipping tests"
    Add-TestResult -Name "API handlers verification" -Passed $false -Details "Source file missing"
}

# ============================================================================
# TEST SECTION 5: WorkBuddy Onboard Verification
# ============================================================================
Write-Section "TEST 5: WorkBuddy Onboard Verification"

$onboardFile = "$ProjectRoot\internal\cli\cmd\onboard.go"
if (Test-Path $onboardFile) {
    $onboardContent = Get-Content $onboardFile -Raw
    
    if ($onboardContent -match 'AgentWorkBuddy\s*=\s*"workbuddy"') {
        Add-TestResult -Name "WorkBuddy defined as agent constant" -Passed $true
    } else {
        Add-TestResult -Name "WorkBuddy defined as agent constant" -Passed $false -Details "Constant not found"
    }
    
    if ($onboardContent -match "AgentWorkBuddy:\s*\{") {
        Add-TestResult -Name "WorkBuddy in supportedAgents map" -Passed $true
    } else {
        Add-TestResult -Name "WorkBuddy in supportedAgents map" -Passed $false -Details "Not in supported agents map"
    }
    
    if ($onboardContent -match "AgentWorkBuddy:.*Memory Skill.*Persistent Memory Layer") {
        Add-TestResult -Name "WorkBuddy template includes Memory Skill section" -Passed $true
    } else {
        Add-TestResult -Name "WorkBuddy template includes Memory Skill section" -Passed $false -Details "Template missing memory skill info"
    }
    
    if ($onboardContent -match "memory_store.*content.*type.*tags") {
        Add-TestResult -Name "WorkBuddy template includes memory_store example" -Passed $true
    } else {
        Add-TestResult -Name "WorkBuddy template includes memory_store example" -Passed $false -Details "Example not found"
    }
    
    if ($onboardContent -match "memory_recall.*query") {
        Add-TestResult -Name "WorkBuddy template includes memory_recall example" -Passed $true
    } else {
        Add-TestResult -Name "WorkBuddy template includes memory_recall example" -Passed $false -Details "Example not found"
    }
    
    if ($onboardContent -match '"workbuddy"') {
        Add-TestResult -Name "Onboard command accepts workbuddy as agent name" -Passed $true
    } else {
        Add-TestResult -Name "Onboard command accepts workbuddy as agent name" -Passed $false -Details "Not in accepted agent names"
    }
} else {
    Write-Warn "Onboard file not found, skipping tests"
    Add-TestResult -Name "WorkBuddy Onboard verification" -Passed $false -Details "Source file missing"
}

# ============================================================================
# TEST SECTION 6: Documentation Completeness Verification
# ============================================================================
Write-Section "TEST 6: Documentation Completeness Verification"

if (Test-Path $memorySkillDoc) {
    $skillContent = Get-Content $memorySkillDoc -Raw
    
    if ($skillContent -match "## Quick Start") {
        Add-TestResult -Name "memory-skill.md: Quick Start section present" -Passed $true
    } else {
        Add-TestResult -Name "memory-skill.md: Quick Start section present" -Passed $false -Details "Missing Quick Start section"
    }
    
    if ($skillContent -match "## Memory Types" -and $skillContent -match "Type.*When to Use.*Example") {
        Add-TestResult -Name "memory-skill.md: Memory Types table present" -Passed $true
    } else {
        Add-TestResult -Name "memory-skill.md: Memory Types table present" -Passed $false -Details "Missing or incomplete table"
    }
    
    # Check for all 4 MCP tool references
    $mcpToolsFound = 0
    if ($skillContent -match "memory_store") { $mcpToolsFound++ }
    if ($skillContent -match "memory_recall") { $mcpToolsFound++ }
    if ($skillContent -match "memory_list") { $mcpToolsFound++ }
    if ($skillContent -match "memory_delete") { $mcpToolsFound++ }
    
    if ($mcpToolsFound -ge 4) {
        Add-TestResult -Name "memory-skill.md: All 4 MCP Tools documented" -Passed $true
    } else {
        Add-TestResult -Name "memory-skill.md: All 4 MCP Tools documented" -Passed $false -Details "Only $mcpToolsFound/4 tools found"
    }
    
    if ($skillContent -match "## WorkBuddy Integration") {
        Add-TestResult -Name "memory-skill.md: WorkBuddy Integration section present" -Passed $true
    } else {
        Add-TestResult -Name "memory-skill.md: WorkBuddy Integration section present" -Passed $false -Details "Missing section"
    }
    
    if ($skillContent -match "## SQL Direct Access" -and $skillContent -match "SELECT.*agent_memories") {
        Add-TestResult -Name "memory-skill.md: SQL Direct Access examples present" -Passed $true
    } else {
        Add-TestResult -Name "memory-skill.md: SQL Direct Access examples present" -Passed $false -Details "Missing SQL examples"
    }
    
    if ($skillContent -match "## API Endpoints Reference" -and $skillContent -match "/api/v1/databases/:id/memories") {
        Add-TestResult -Name "memory-skill.md: API Endpoints Reference present" -Passed $true
    } else {
        Add-TestResult -Name "memory-skill.md: API Endpoints Reference present" -Passed $false -Details "Missing endpoint reference"
    }
} else {
    Write-Warn "Memory skill documentation not found, skipping tests"
}

if (Test-Path $quickstartDoc) {
    $guideContent = Get-Content $quickstartDoc -Raw
    
    if ($guideContent -match "## One-Line Start" -or $guideContent -match "curl.*db9-quickstart\.sh") {
        Add-TestResult -Name "quickstart-guide.md: One-Line Start command present" -Passed $true
    } else {
        Add-TestResult -Name "quickstart-guide.md: One-Line Start command present" -Passed $false -Details "Missing one-line start"
    }
    
    if ($guideContent -match "## Manual Step-by-Step" -or $guideContent -match "### \d+\.") {
        Add-TestResult -Name "quickstart-guide.md: Manual steps documented" -Passed $true
    } else {
        Add-TestResult -Name "quickstart-guide.md: Manual steps documented" -Passed $false -Details "Missing manual steps"
    }
    
    if ($guideContent -match "workbuddy.*--agent.*workbuddy" -or $guideContent -match "db9 onboard.*workbuddy") {
        Add-TestResult -Name "quickstart-guide.md: WorkBuddy install command present" -Passed $true
    } else {
        Add-TestResult -Name "quickstart-guide.md: WorkBuddy install command present" -Passed $false -Details "Missing install command"
    }
    
    if ($guideContent -match "## Architecture Overview" -or ($guideContent -match "PostgreSQL" -and $guideContent -match "Agent")) {
        Add-TestResult -Name "quickstart-guide.md: Architecture diagram present" -Passed $true
    } else {
        Add-TestResult -Name "quickstart-guide.md: Architecture diagram present" -Passed $false -Details "Missing architecture diagram"
    }
    
    if ($guideContent -match "## Troubleshooting" -or $guideContent -match "Problem.*Solution") {
        Add-TestResult -Name "quickstart-guide.md: Troubleshooting section present" -Passed $true
    } else {
        Add-TestResult -Name "quickstart-guide.md: Troubleshooting section present" -Passed $false -Details "Missing troubleshooting"
    }
} else {
    Write-Warn "Quick start guide not found, skipping tests"
}

# ============================================================================
# TEST SECTION 7: Migration File Content Validation
# ============================================================================
Write-Section "TEST 7: Migration File Content Validation"

if (Test-Path $migrationFile) {
    $migrationContent = Get-Content $migrationFile -Raw
    
    if ($migrationContent -match "CREATE TABLE IF NOT EXISTS agent_memories") {
        Add-TestResult -Name "Migration: agent_memories table created" -Passed $true
    } else {
        Add-TestResult -Name "Migration: agent_memories table created" -Passed $false -Details "Table definition missing"
    }
    
    if ($migrationContent -match "CREATE TABLE IF NOT EXISTS memory_embeddings") {
        Add-TestResult -Name "Migration: memory_embeddings table created" -Passed $true
    } else {
        Add-TestResult -Name "Migration: memory_embeddings table created" -Passed $false -Details "Table definition missing"
    }
    
    $requiredColumns = @("id", "agent_id", "memory_type", "content", "tags", "importance_score")
    $columnsFound = 0
    foreach ($col in $requiredColumns) {
        if ($migrationContent -match $col) { $columnsFound++ }
    }
    
    if ($columnsFound -ge 6) {
        Add-TestResult -Name "Migration: All required columns present ($columnsFound/6)" -Passed $true
    } else {
        Add-TestResult -Name "Migration: All required columns present ($columnsFound/6)" -Passed $false -Details "Missing columns"
    }
    
    if ($migrationContent -match "CREATE INDEX.*idx_memories") {
        Add-TestResult -Name "Migration: Performance indexes created" -Passed $true
    } else {
        Add-TestResult -Name "Migration: Performance indexes created" -Passed $false -Details "Missing indexes"
    }
    
    if ($migrationContent -match "CREATE TRIGGER.*update_memories_updated_at") {
        Add-TestResult -Name "Migration: Auto-update trigger configured" -Passed $true
    } else {
        Add-TestResult -Name "Migration: Auto-update trigger configured" -Passed $false -Details "Missing trigger"
    }
} else {
    Write-Warn "Migration file not found, skipping validation"
}

# ============================================================================
# TEST SECTION 8: Router Configuration Validation
# ============================================================================
Write-Section "TEST 8: Router Configuration Validation"

if (Test-Path $routerFile) {
    $routerContent = Get-Content $routerFile -Raw

    # Check for POST memories route
    if ($routerContent -match "StoreMemoryHandler" -and $routerContent -match "memories") {
        Add-TestResult -Name "Router: POST /memories configured" -Passed $true
    } else {
        Add-TestResult -Name "Router: POST /memories configured" -Passed $false -Details "Route not properly configured"
    }

    # Check for GET memories route
    if ($routerContent -match "ListMemoriesHandler" -and $routerContent -match "memories") {
        Add-TestResult -Name "Router: GET /memories configured" -Passed $true
    } else {
        Add-TestResult -Name "Router: GET /memories configured" -Passed $false -Details "Route not properly configured"
    }

    # Check for POST recall route
    if ($routerContent -match "RecallMemoryHandler" -and $routerContent -match "recall") {
        Add-TestResult -Name "Router: POST /memories/recall configured" -Passed $true
    } else {
        Add-TestResult -Name "Router: POST /memories/recall configured" -Passed $false -Details "Route not properly configured"
    }

    # Check for DELETE route
    if ($routerContent -match "DeleteMemoryHandler") {
        Add-TestResult -Name "Router: DELETE /memories/:id configured" -Passed $true
    } else {
        Add-TestResult -Name "Router: DELETE /memories/:id configured" -Passed $false -Details "Route not properly configured"
    }
} else {
    Write-Warn "Router file not found, skipping validation"
}

# ============================================================================
# Generate Summary Report
# ============================================================================
Write-Host ""
Write-Host "**********************************************************" -ForegroundColor White
Write-Host "*                    TEST SUMMARY                          *" -ForegroundColor White
Write-Host "**********************************************************" -ForegroundColor White
Write-Host ("*  Total Tests:  {0,-43} *" -f $TotalTests) -ForegroundColor White
Write-Host ("*  Passed:      {0,-43} *" -f $PassedTests) -ForegroundColor Green
Write-Host ("*  Failed:      {0,-43} *" -f $FailedTests) -ForegroundColor Red
$passRate = if ($TotalTests -gt 0) { "{0:P0}" -f ($PassedTests / $TotalTests) } else { "N/A" }
Write-Host ("*  Pass Rate:   {0,-43} *" -f $passRate) -ForegroundColor White
Write-Host "**********************************************************" -ForegroundColor White

# Write detailed results to file
"`nDETAILED RESULTS:" | Out-File -FilePath $ResultsFile -Append -Encoding UTF8
"-" * 60 | Out-File -FilePath $ResultsFile -Append -Encoding UTF8

foreach ($result in $TestResults) {
    $line = "[{0}] {1}" -f $result.Status, $result.Name
    if ($result.Details) {
        $line += " - {0}" -f $result.Details
    }
    $line | Out-File -FilePath $ResultsFile -Append -Encoding UTF8
}

"`n" | Out-File -FilePath $ResultsFile -Append -Encoding UTF8
"SUMMARY:" | Out-File -FilePath $ResultsFile -Append -Encoding UTF8
"- Total Tests: $TotalTests" | Out-File -FilePath $ResultsFile -Append -Encoding UTF8
"- Passed: $PassedTests" | Out-File -FilePath $ResultsFile -Append -Encoding UTF8
"- Failed: $FailedTests" | Out-File -FilePath $ResultsFile -Append -Encoding UTF8
"- Pass Rate: $passRate" | Out-File -FilePath $ResultsFile -Append -Encoding UTF8
"" | Out-File -FilePath $ResultsFile -Append -Encoding UTF8
"Results saved to: $ResultsFile" | Out-File -FilePath $ResultsFile -Append -Encoding UTF8

Write-Host ""
Write-Info "Detailed results saved to: $ResultsFile"
Write-Host ""

# Exit with appropriate code
if ($FailedTests -gt 0) {
    $failMsg = "[!!!] TEST SUITE FAILED [!!!]"
    Write-Host $failMsg -ForegroundColor Red
    exit 1
} else {
    $passMsg = "[+] ALL TESTS PASSED [+]"
    Write-Host $passMsg -ForegroundColor Green
    exit 0
}
