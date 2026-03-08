# Event Bus 001 Plan 03: 锁租约与冲突控制

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 建立 `lock.md` 的租约协议，让 orchestrator、human-gate、recovery 等未来写入方可以通过版本化 lease 做互斥写操作。

**Architecture:** 将 `lock.md` 视为 event-bus 的控制平面状态文件，使用原子替换写和版本号比较做 acquire / renew / release。锁协议独立于业务事件 WAL，不把锁自举问题复杂化到 `events.md` 中。

**Tech Stack:** Go、标准库

---

### Task 1: 实现锁状态读取与 acquire

**Files:**
- Create: `internal/eventbus/lock.go`
- Test: `internal/eventbus/lock_acquire_test.go`
- Reference: `docs/event-bus/001/prd.md`

**Step 1: 写失败测试，锁定初始读取与首次获取**

- bootstrap 后读取 `lock.md`，断言 `status=FREE`。
- 首次 acquire 成功后，断言 `holder`、`lease_token`、`lease_version`、`expire_at` 都被写入。

Run: `go test ./internal/eventbus -run TestAcquireLock -v`
Expected: 先失败，说明锁读取和获取尚未完成。

**Step 2: 实现 `ReadLock()` 与 `AcquireLock()`**

- 识别 `FREE` / `HELD` / `EXPIRED` 三种状态。
- 生成 lease token。
- 递增 `lease_version` 并更新过期时间。

**Step 3: 回跑首次获取测试**

Run: `go test ./internal/eventbus -run TestAcquireLock -v`
Expected: 通过。

### Task 2: 实现 renew / release 与版本冲突

**Files:**
- Modify: `internal/eventbus/lock.go`
- Test: `internal/eventbus/lock_lifecycle_test.go`
- Reference: `docs/event-bus/001/e2e.md`

**Step 1: 写失败测试，锁定版本约束**

- 使用正确 `lease_token + lease_version` 续租成功。
- 使用旧版本或错误 token 续租 / 释放失败。
- 失败后 `lock.md` 保持原状态。

Run: `go test ./internal/eventbus -run TestLockLifecycle -v`
Expected: 先失败，说明版本冲突控制尚未建立。

**Step 2: 实现 `RenewLock()` 与 `ReleaseLock()`**

- 同时校验 token 和 version。
- 成功时递增 `lease_version`。
- 释放后回到 `FREE` 状态，但保留最近操作痕迹。

**Step 3: 回跑锁生命周期测试**

Run: `go test ./internal/eventbus -run TestLockLifecycle -v`
Expected: 通过。

### Task 3: 实现过期判定与错误归一化

**Files:**
- Modify: `internal/eventbus/lock.go`
- Test: `internal/eventbus/lock_expire_test.go`
- Reference: `docs/event-bus/001/prd.md`

**Step 1: 写失败测试，锁定过期重入行为**

- 让一个 lease 过期后，另一个 holder 可以重新 acquire。
- 断言已过期 lease 后续 renew / release 返回 `lease expired`。

Run: `go test ./internal/eventbus -run TestLockExpire -v`
Expected: 先失败，说明过期判定行为尚未稳定。

**Step 2: 实现过期计算和错误归一化**

- 统一使用 UTC 时间。
- 把版本冲突、token 不匹配、lease 过期映射为稳定错误类型。
- 不依赖错误字符串做流程分支。

**Step 3: 回跑 lock 全量测试**

Run: `go test ./internal/eventbus -run TestLock -v`
Expected: 锁相关测试全部通过。
