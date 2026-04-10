# Enhance Memory Skill Guidance for AI Agents Spec

## Why

当前的 SKILL.md 虽然包含了基本的使用方法（如何存储、检索、列出、删除记忆），但缺少关键的**使用策略和触发时机指引**，导致 AI Agent（如 WorkBuddy）无法判断：

* **何时应该存储记忆** vs 何时不需要

* **存储什么内容**最有价值

* **如何组织记忆**以最大化检索效率

* **在对话的哪个阶段**执行记忆操作

这导致 Agent 要么过度存储（噪音过多），要么遗漏重要信息（信息丢失）。需要添加更细致的、可操作的决策框架。

## What Changes

### 1. 新增 "触发时机与决策框架" 章节 (Trigger Timing & Decision Framework)

* 定义明确的触发条件（何时调用 memory\_store）

* 提供决策树/流程图帮助 Agent 判断是否值得存储

* 区分不同类型信息的处理优先级

### 2. 新增 "智能过滤与去重策略" 章节 (Smart Filtering & Deduplication)

* 如何避免重复存储相似内容

* 内容摘要 vs 原文存储的策略

* 记忆生命周期管理（何时更新、归档、删除）

### 3. 新增 "上下文感知使用模式" 章节 (Context-Aware Usage Patterns)

* 对话开始时：快速恢复上下文（memory\_recall）

* 对话进行中：选择性存储关键信息（memory\_store）

* 对话结束时：总结和整理记忆（memory\_list + memory\_store）

* 长期维护：定期清理和优化（SQL 分析 + memory\_delete）

### 4. 增强 WorkBuddy 特定优化章节

* 针对 WorkBuddy/OpenClaw 的最佳实践

* 与其他工具链的集成建议

* 性能优化技巧

### 5. 添加实用示例库 (Example Library)

* 10+ 个真实场景的完整示例（含触发条件说明）

* 好的实践 vs 差的实践对比

* 常见反模式和解决方案

## Impact

### Affected specs:

* fix-workbuddy-skill-structure (增强 SKILL.md 内容)

### Affected code:

* `skills/db9/SKILL.md` - 主要修改目标（增强现有文档）

* `WORKBUDDY_PROMPT.md` - 同步更新关键指引

* `internal/cli/cmd/onboard.go` - 可能需要更新模板（如果新增示例）

## ADDED Requirements

### Requirement: Trigger Timing Decision Framework

The system SHALL provide clear, actionable criteria for when to store memories.

#### Scenario: User mentions a preference

* **WHEN** user says "I prefer using TypeScript" or similar preference statement

* **THEN** agent SHALL:

  1. Extract the core preference (language=TypeScript, context=frontend/fullstack)
  2. Check if similar memory exists (using memory\_recall with query="user preferences")
  3. If not exists or significantly different → store as `type="preference"` with importance\_score >= 0.8
  4. If exists and similar → update existing memory (delete old + store new) or skip

#### Scenario: User makes a technical decision

* **WHEN** user chooses a specific technology/architecture/pattern

* **THEN** agent SHALL store immediately as `type="decision"` with importance\_score >= 0.9 AND include reasoning/context

#### Scenario: Error or failure occurs

* **WHEN** an error is encountered and resolved (or lesson learned)

* **THEN** agent SHALL store as `type="error"` with:

  * The error description

  * Root cause (if known)

  * Solution that worked

  * Prevention tips

### Requirement: Content Value Assessment Criteria

The system SHALL help agents assess whether content is worth storing.

#### Scenario: Evaluating information value

Agent SHALL use this checklist before storing:

1. **Is it reusable?** Will this be useful in future sessions? (Yes → Store)
2. **Is it specific enough?** Vague info like "good project" → Skip or summarize
3. **Is it already stored?** Check via recall first to avoid duplicates
4. **Is it actionable?** Can it influence future decisions? (Yes → High importance)
5. **Will it expire soon?** Temporary info (debug output, temp files) → Low importance or skip

### Requirement: Context-Aware Usage Patterns

The system SHALL define clear patterns for different conversation phases.

#### Pattern 1: Session Start (Context Recovery)

```
TRIGGER: New conversation session starts
ACTION: 
  1. Call memory_recall(query="recent decisions and context", top_k=10)
  2. Call memory_recall(query="current project status", top_k=5)
  3. Summarize findings for user: "Based on our previous work..."
```

#### Pattern 2: During Conversation (Selective Capture)

```
TRIGGER: User shares important info OR makes decision OR encounters error
ACTION:
  1. Evaluate against value criteria (see above)
  2. If worth storing → call memory_store with appropriate type/tags/score
  3. Continue conversation without interruption
```

#### Pattern 3: Session End (Consolidation)

```
TRIGGER: User says goodbye / task completed / session timeout
ACTION:
  1. Call memory_list to review session's memories
  2. Identify key outcomes and open items
  3. Store summary as type="observation" or type="plan"
  4. Clean up temporary/low-value memories
```

### Requirement: Anti-Patterns and Common Mistakes

The system SHALL document common mistakes to avoid.

#### Anti-Pattern #1: Over-Storage

* ❌ Storing every user message

* ❌ Storing trivial facts ("user said ok")

* ✅ Only store: preferences, decisions, errors, key facts, plans

#### Anti-Pattern #2: Under-Storage

* ❌ Missing important decisions

* ❌ Forgetting error lessons

* ✅ When in doubt, store it (you can delete later)

#### Anti-Pattern #3: Poor Organization

* ❌ Using generic tags like "info", "misc"

* ❌ Not setting importance scores

* ✅ Use specific tags: "frontend", "typescript", "critical"

* ✅ Set scores: decisions=0.9+, preferences=0.8+, observations=0.5

## MODIFIED Requirements

### Requirement: SKILL.md Structure (from fix-workbuddy-skill-structure)

The SKILL.md SHALL be enhanced with new chapters:

1. **Current Chapter 6 (Best Practices)** → Expand with trigger timing guidance
2. **New Chapter X: 触发时机与决策框架 (When to Store Memories)**

   * Decision tree diagram

   * Value assessment checklist

   * Priority matrix
3. **New Chapter Y: 上下文感知使用模式 (Usage Patterns by Phase)**

   * Session start/mid/end patterns

   * Real-time decision flowchart
4. **New Chapter Z: 实战示例库 (Real-World Example Library)**

   * 10+ categorized examples

   * Before/after comparisons

   * Trigger condition annotations

## Implementation Notes

### Key Design Principles

1. **Actionable**: Every guideline should be testable and implementable
2. **Specific**: Avoid vague advice like "store important things"; instead say "store when user explicitly states a preference"
3. **Balanced**: Prevent both over-storage and under-storage
4. **Adaptable**: Allow agents to customize thresholds based on use case

### Example Format Enhancement

Each example SHOULD include:

```markdown
### Example: [Title]
**Trigger Condition:** [Specific condition that triggers storage]
**User Input:** [What user said/did]
**Agent Action:** [Exact API call with parameters]
**Why This Matters:** [Explanation of value]
**Alternative (Bad):** [What NOT to do and why]
```

