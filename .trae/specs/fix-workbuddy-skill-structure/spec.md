# Fix WorkBuddy Skill Structure & Encoding Spec

## Why
WorkBuddy 的 Skills 系统要求使用 **目录 + SKILL.md 文件** 的标准结构，而非单个 .md 文件。同时，PowerShell 的 JSON 请求默认使用系统编码，导致中文内容乱码。这两个问题严重影响了用户体验和功能可用性。

## What Changes

### 1. **BREAKING**: 修改 Skills 目录结构
- 将 `skills/memory-skill.md` (单文件) 重构为 `skills/db9/` 目录结构
- 创建标准的 `SKILL.md` 文件（含 YAML frontmatter）
- 删除旧的 `memory-skill.md` 和 `quickstart-guide.md`

### 2. 修复 Onboard 命令安装路径
- 更新 `internal/cli/cmd/onboard.go` 中 WorkBuddy 的安装路径
- 从 `.workbuddy/skills/db9.md` 改为 `.workbuddy/skills/db9/SKILL.md`
- 安装时自动创建目录结构

### 3. 添加 UTF-8 编码支持文档
- 在 SKILL.md 中添加 PowerShell UTF-8 编码注意事项
- 提供正确的编码示例代码
- 在 WORKBUDDY_PROMPT.md 中强调编码问题

## Impact

### Affected specs:
- agent-memory-skill (更新 checklist 和 tasks)

### Affected code:
- `internal/cli/cmd/onboard.go` - 修改 WorkBuddy 安装路径和模板
- `skills/` 目录 - 重构为标准目录结构
- `WORKBUDDY_PROMPT.md` - 添加编码注意事项

## ADDED Requirements

### Requirement: Standard Skill Directory Structure
The system SHALL use the standard WorkBuddy skill structure: `~/.workbuddy/skills/<skill-name>/SKILL.md`

#### Scenario: Install DB9 skill for WorkBuddy
- **WHEN** user runs `db9 onboard --agent workbuddy`
- **THEN** system creates directory `~/.workbuddy/skills/db9/`
- **AND** creates file `~/.workbuddy/skills/db9/SKILL.md` with standard frontmatter
- **AND** skill is discoverable in WorkBuddy's skills search

#### Scenario: SKILL.md contains proper frontmatter
- **WHEN** user opens the installed SKILL.md
- **THEN** file starts with YAML metadata block
- **AND** includes name, description, version, author fields
- **AND** followed by comprehensive usage documentation

### Requirement: UTF-8 Encoding Support for Chinese Content
The system SHALL provide clear guidance on handling non-ASCII content in API requests.

#### Scenario: Store memory with Chinese content via PowerShell
- **WHEN** user stores memory containing Chinese characters using PowerShell
- **THEN** documentation shows correct UTF-8 encoding method: `[System.Text.Encoding]::UTF8.GetBytes($body)`
- **AND** content is stored correctly without garbled characters
- **AND** retrieval returns original Chinese text intact

#### Scenario: Recall memories with Chinese queries
- **WHEN** user searches memories using Chinese query string
- **THEN** semantic search works correctly with CJK characters
- **AND** results contain properly encoded Chinese content

## MODIFIED Requirements

### Requirement: WorkBuddy Agent Template (from agent-memory-skill)
The WorkBuddy onboard template SHALL be updated to:

1. Use correct installation path (`skills/db9/SKILL.md`)
2. Include complete SKILL.md template with frontmatter
3. Add UTF-8 encoding section for PowerShell users
4. Provide working examples with Chinese content
5. Include troubleshooting guide for encoding issues

---

## Implementation Plan

### Task 1: Restructure skills/ directory
- [ ] Create `skills/db9/` directory
- [ ] Create `skills/db9/SKILL.md` with full content and frontmatter
- [ ] Move relevant content from `memory-skill.md` to new SKILL.md
- [ ] Delete old `memory-skill.md` and `quickstart-guide.md`

### Task 2: Update onboard.go
- [ ] Change WorkBuddy path from `.workbuddy/skills/db9.md` to `.workbuddy/skills/db9/SKILL.md`
- [ ] Update install logic to create directory if not exists
- [ ] Update WorkBuddy template with new SKILL.md format
- [ ] Add UTF-8 encoding examples in template

### Task 3: Update WORKBUDDY_PROMPT.md
- [ ] Add "Known Issues" section about encoding
- [ ] Provide PowerShell UTF-8 examples
- [ ] Add troubleshooting steps for garbled characters

### Task 4: Verification
- [ ] Test `db9 onboard --agent workbuddy` creates correct structure
- [ ] Verify SKILL.md has valid frontmatter
- [ ] Test Chinese content storage/retrieval
- [ ] Run existing E2E tests (56 tests) still pass
