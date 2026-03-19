# AGENTS.md — BlendPOS Code Review Rules

**Used by:** Gentleman Guardian Angel (gga) | **Scope:** TypeScript/React | **Goal:** Enforce BlendPOS patterns before merge.

---

## Severity Levels

| 🔴 **FAIL** | 🟡 **WARN** | 🔵 **INFO** |
|---|---|---|
| Block merge | Risky; needs approval | Best practice nudge |

---

## TypeScript Type Safety

### ✋ FAIL: `any` Type Usage
```typescript
// ❌ const data: any = response.data;
// ✅ const data: Sale = response.data as Sale;
```
Exception: Legacy 3rd-party types (comment WHY).

### ✋ FAIL: Loose Object Shapes
```typescript
// ❌ interface CartStore { data: Record<string, any> }
// ✅ interface CartStore { items: CartItem[]; totals: CartTotals }
```

---

## Zustand State Management

### ✋ FAIL: Missing Store Type Export
```typescript
// ❌ export const useCartStore = create<CartStore>(...)
// ✅ export type CartStore = { ... }; export const useCartStore = ...
```

### 🟡 WARN: Selector Recreations
```typescript
// ❌ useCartStore((s) => s.items.filter(...))  // new object every render
// ✅ useCartStore(useShallow((s) => ({ items: s.items })))
```

### 🔵 INFO: Store Mutation Logic
Complex mutations belong in store methods, not components:
```typescript
// Store: addItem: (item) => set((s) => ({ items: [...s.items, item] }))
// Component: useCartStore.getState().addItem(item);
```

---

## Offline-First / Dexie.js

### ✋ FAIL: Missing `offline_id` on Sync
```typescript
// ❌ await db.sales.add(sale);  // No dedup!
// ✅ const id = crypto.randomUUID(); await db.sales.add({ ...sale, offline_id: id });
```

### ✋ FAIL: Dexie Query Without Index
**Schema must include indexed field:**
```typescript
db.version(1).stores({
  sales: '++id, offline_id, tenantID'  // ← tenantID indexed
});
db.sales.where('tenantID').equals(tid).toArray();  // ✅
```

### 🟡 WARN: Delete Without Sync Queue
```typescript
// ✅ await queueForSync({ action: 'delete', id: saleID }); 
// ✅ await db.sales.delete(saleID);  // then local
```

---

## React Components

### ✋ FAIL: Hardcoded API Endpoints
```typescript
// ❌ fetch('http://localhost:8000/v1/sales');
// ✅ import { salesAPI } from 'src/api/client'; salesAPI.create(sale);
```

### ✋ FAIL: Direct DOM Access
```typescript
// ❌ useEffect(() => { const el = document.getElementById('printer'); });
// ✅ const ref = useRef(null); useEffect(() => { ref.current?.focus(); });
```

### 🟡 WARN: Missing Loading / Error States
```typescript
// ✅ Show <Spinner /> while loading, <Alert /> on error, content when ready
```

### 🟡 WARN: No Error Boundary
```typescript
// ✅ Wrap data-fetching pages in <ErrorBoundary>
```

---

## Naming Conventions

### 🔵 INFO: Hook Naming
```typescript
// ✅ const useCartStore = create(...)  // use + Name + Store
// ✅ const handleAddItem = () => {}    // handle{Action}
```

### 🟡 WARN: Prop Drilling > 3 Levels
```typescript
// ✅ Use store: const tenantID = useAuthStore((s) => s.tenantID);
```

---

## Testing

### 🟡 WARN: Missing Async Tests
```typescript
// ✅ Test API calls, store mutations, side effects
```

### 🔵 INFO: Behavior Over Snapshots
```typescript
// ✅ expect(getByRole('button')).toHaveTextContent('Click');
// ❌ expect(render(...)).toMatchSnapshot();
```

---

## Performance

### 🟡 WARN: Inline Objects/Arrays
```typescript
// ❌ <Component options={['a', 'b']} />
// ✅ const OPTIONS = ['a', 'b']; <Component options={OPTIONS} />
```

### 🟡 WARN: Missing `key` in Lists
```typescript
// ❌ items.map((item, i) => <Item key={i} />)
// ✅ items.map((item) => <Item key={item.id} />)
```

---

## Tenant Context / Multi-Tenancy ⭐

### ✋ FAIL: Missing Tenant Filter
```typescript
// ❌ db.sales.toArray();  // ALL tenants! Security breach.
// ✅ const tid = useAuthStore((s) => s.tenantID); db.sales.where('tenantID').equals(tid).toArray();
```

### 🟡 WARN: Hardcoded Tenant ID
```typescript
// ✅ Extract from auth store: const tenantID = useAuthStore((s) => s.tenantID);
// ❌ Never hardcode: db.sales.where('tenantID').equals('tenant-123')
```

---

## Exception: Skip Rule

```typescript
// @agent-skip:any_type - legacy API response
const data: any = response;
```

---

## Quick Reference

| Rule | Severity |
|------|----------|
| No `any` | FAIL |
| Store type export | FAIL |
| `offline_id` on sync | FAIL |
| **Tenant filter queries** | **FAIL** ⭐ |
| Hardcoded API endpoints | FAIL |
| Loading/error states | WARN |
| Selector memoization | WARN |
| Prop drilling > 3 | WARN |
| Missing async tests | WARN |

---

## Integration (GGA Config)

`.gga` file:
```
RULES_FILE="AGENTS.md"
FILE_PATTERNS="*.ts,*.tsx,*.js,*.jsx"
EXCLUDE_PATTERNS="*.test.ts,*.spec.ts,*.test.tsx,*.spec.tsx,*.d.ts"
```

Usage:
```bash
gga review --pr {PR_NUMBER}  # Review PR
gga review src/               # Review folder
```

---

## Context

This review suite enforces the **SaaS Multi-Tenant** architecture currently in progress.  
See `/openspec/changes/saas-multi-tenant/` for specs, design, and implementation tasks.

**Key principle:** Tenant isolation is a SECURITY requirement, not an optional best practice.  
Every filter matters. Every query needs `tenantID`.
