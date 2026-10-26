# cache

[English](README.md)

面向 Go 的小型多后端字节缓存。

- **契约很薄。** `Store` 只有 `Get` / `Set` / `Delete`，值为 `[]byte`，未命中返回 `ErrMiss`。
- **需要时再加门面。** `Cache` 提供默认 TTL、键前缀和 `GetOrSet`；业务也可以直接用 `Store`。
- **驱动可选。** 核心模块无网络依赖；驱动是独立 module，按需引入。
- **基础设施归应用。** NATS KV 桶与 Postgres 表由宿主创建与迁移；驱动不 `CREATE`。

需要 Go 1.27+。

## 安装

```sh
go get github.com/runforyou-ai/cache
go get github.com/runforyou-ai/cache/nats   # 可选
go get github.com/runforyou-ai/cache/pg    # 可选
```

## 用法

```go
import (
	"github.com/runforyou-ai/cache"
	"github.com/runforyou-ai/cache/memory"
)

store := memory.New()
c := cache.New(store, cache.WithDefaultTTL(time.Hour), cache.WithPrefix("app:"))

value, err := c.GetOrSet(ctx, "route", cache.NoExpiration, func(ctx context.Context) ([]byte, error) {
	return load(), nil
})
```

### NATS JetStream KV

注入已准备好的桶（缓存场景通常 History=1，并设置桶级 TTL）：

```go
import natscache "github.com/runforyou-ai/cache/nats"

store := natscache.New(kv) // kv 为 jetstream.KeyValue
// Set 上的 per-entry TTL 会被忽略，过期依赖桶的 MaxAge。
```

### PostgreSQL（UNLOGGED）

通过应用迁移执行 [`pg/schema.sql`](pg/schema.sql)，然后：

```go
import "github.com/runforyou-ai/cache/pg"

store := pg.Open(pool) // *pgxpool.Pool 或任意 pg.DB
// 可选：pg.WithTable("app_cache")、pg.WithGCProbability(100)
```

推荐使用 `UNLOGGED` 表：不写 WAL、非正常崩溃后表被清空、不复制到 standby，符合「可丢弃」的缓存语义。

## 驱动

| 包 | 模块 | 说明 |
|----|------|------|
| `cache/memory` | 核心 | 进程内；可选后台清理 |
| `cache/nats` | `cache/nats` | JetStream KV；注入已有桶 |
| `cache/pg` | `cache/pg` | Postgres 表；schema 由应用管理 |
| `cache.Null` | 核心 | 始终未命中；干净地关掉缓存 |

`cache/storetest` 是各驱动共用的契约测试。

## 许可证

MIT
