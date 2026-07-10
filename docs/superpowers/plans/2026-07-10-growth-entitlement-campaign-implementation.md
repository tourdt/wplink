# 新手成长权益活动 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现可配置、可暂停、可停用的新手成长权益活动，用真实发布和有效分享替代“完善商家名片得更多发布次数”的冷启动激励。

**Architecture:** 新增增长活动、活动规则和发放记录三类数据；后端通过统一增长事件入口匹配启用规则，并复用 `merchant_entitlements` 发放 `publish_quota`、`refresh_quota`、`top_voucher`。后台提供活动与规则配置，小程序只做发布、分享和权益页的轻提示。

**Tech Stack:** Go 1.23、go-zero/goctl、PostgreSQL、`database/sql`、Vue 3、Element Plus、uni-app、Node.js `node:test`、Go test。

---

## File Structure

- Create: `backend/migrations/000020_growth_campaigns.up.sql`
  - 新增 `growth_campaigns`、`growth_campaign_rules`、`growth_reward_grants`。
- Create: `backend/migrations/000020_growth_campaigns.down.sql`
  - 回滚三张增长活动表。
- Modify: `backend/scripts/validate_migrations.test.mjs`
  - 增加增长活动迁移静态校验。
- Create: `backend/app/internal/model/growth_campaign_model.go`
  - 承载活动配置、规则匹配、幂等发放、后台列表和保存方法。
- Create: `backend/app/internal/model/growth_campaign_model_test.go`
  - 校验 source type、幂等键、活动停用和发放 SQL 语义。
- Modify: `backend/app/internal/svc/service_context.go`
  - 把 `GrowthCampaignModel` 挂到 `APIStore`。
- Create: `backend/app/internal/logic/growth/growth_reward_logic.go`
  - 提供统一事件入口，负责非阻塞日志和友好错误边界。
- Create: `backend/app/internal/logic/growth/growth_reward_logic_test.go`
  - 校验事件输入、跳过策略和失败日志不阻断业务。
- Modify: `backend/app/internal/logic/auth/auth_logic.go`
  - 登录成功并确保默认商家后触发 `user_first_login`。
- Modify: `backend/app/internal/logic/auth/auth_logic_test.go`
  - 校验登录触发新手权益事件且失败不影响登录。
- Modify: `backend/app/internal/logic/admin/review_resource_logic.go`
  - 资源审核通过后触发 `resource_first_approved` 和 `resource_approved_count_reached`。
- Modify: `backend/app/internal/logic/admin/review_resource_logic_test.go`
  - 校验审核通过触发事件，驳回和下架不触发。
- Modify: `backend/app/internal/logic/metrics/record_contact_logic.go`
  - `phone`、`wechat`、`share_view` 成功记录后触发分享类成长事件。
- Modify: `backend/app/internal/logic/metrics/record_contact_logic_test.go`
  - 校验有效联系触发奖励，商家看自己资源不触发。
- Modify: `backend/app/internal/model/resource_model.go`
  - 移除按 `profile_status` 区分普通月度发布额度的隐性奖励。
- Modify: `backend/app/internal/model/resource_model_test.go`
  - 更新普通用户额度测试，确认不再因商家名片完善获得更多发布次数。
- Modify: `backend/app/internal/server/domain_routes.go`
  - 增加后台增长活动配置接口和 `share_view` 联系事件支持。
- Modify: `backend/app/internal/server/remaining_api_test.go`
  - 增加后台接口权限测试。
- Create: `backend/app/internal/logic/admin/growth_campaign_logic.go`
  - 后台活动、规则、发放记录的列表和保存逻辑。
- Create: `backend/app/internal/logic/admin/growth_campaign_logic_test.go`
  - 校验奖励数量、有效期、状态流转和停用原因。
- Create: `admin-web/src/api/growthCampaign.js`
  - 后台增长活动 API 封装。
- Create: `admin-web/src/views/GrowthCampaignView.vue`
  - 活动列表、规则编辑、暂停、停用和发放记录。
- Modify: `admin-web/src/router/index.js`
  - 增加 `/growth-campaigns` 路由。
- Modify: `admin-web/src/layouts/AdminLayout.vue`
  - 增加“增长活动”菜单。
- Create: `admin-web/scripts/growth-campaign-view.test.mjs`
  - 静态校验后台页面包含关键控件和停用文案。
- Modify: `wxapp/common/resourceShare.js`
  - 分享路径追加分享来源参数，支持后续有效浏览归因。
- Modify: `wxapp/common/resourceShare.test.mjs`
  - 校验分享路径带资源 ID 和来源参数。
- Modify: `wxapp/pages/resource/detail.vue`
  - 从分享来源进入时记录 `share_view`，保留当前分享封面能力。
- Modify: `wxapp/pages/resource/detail.test.mjs`
  - 校验 `share_view` 只在分享来源存在时记录。
- Modify: `wxapp/pages/vip/index.vue` 或现有权益展示入口
  - 文案增加成长权益来源，不强调商家名片。
- Modify: `wxapp/pages/vip/index.test.mjs`
  - 校验权益文案不包含“完善商家名片得更多发布次数”。

## Task 1: 数据库迁移

**Files:**
- Create: `backend/migrations/000020_growth_campaigns.up.sql`
- Create: `backend/migrations/000020_growth_campaigns.down.sql`
- Modify: `backend/scripts/validate_migrations.test.mjs`

- [ ] **Step 1: Write failing migration test**

在 `backend/scripts/validate_migrations.test.mjs` 追加测试：

```js
test('growth campaign migration supports configurable stoppable rewards', () => {
  const upSql = fs.readFileSync(path.resolve(migrationsDir, '000020_growth_campaigns.up.sql'), 'utf8')
  const downSql = fs.readFileSync(path.resolve(migrationsDir, '000020_growth_campaigns.down.sql'), 'utf8')

  for (const snippet of [
    'CREATE TABLE IF NOT EXISTS growth_campaigns',
    'CREATE TABLE IF NOT EXISTS growth_campaign_rules',
    'CREATE TABLE IF NOT EXISTS growth_reward_grants',
    "status varchar(32) NOT NULL DEFAULT 'draft'",
    "source_type",
    "idempotency_key varchar(255) NOT NULL",
    'UNIQUE (idempotency_key)',
    'idx_growth_campaign_rules_campaign_status',
    'idx_growth_reward_grants_merchant',
  ]) {
    assert(upSql.includes(snippet), `growth campaign migration should include snippet ${snippet}`)
  }

  for (const snippet of [
    'DROP TABLE IF EXISTS growth_reward_grants',
    'DROP TABLE IF EXISTS growth_campaign_rules',
    'DROP TABLE IF EXISTS growth_campaigns',
  ]) {
    assert(downSql.includes(snippet), `growth campaign rollback should include snippet ${snippet}`)
  }
})
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
node --test backend/scripts/validate_migrations.test.mjs
```

Expected: FAIL because `000020_growth_campaigns.up.sql` and `.down.sql` do not exist.

- [ ] **Step 3: Add migration SQL**

Create `backend/migrations/000020_growth_campaigns.up.sql`:

```sql
CREATE TABLE IF NOT EXISTS growth_campaigns (
  id bigint PRIMARY KEY DEFAULT next_tsid(),
  code varchar(64) NOT NULL UNIQUE,
  name varchar(128) NOT NULL,
  status varchar(32) NOT NULL DEFAULT 'draft',
  starts_at timestamptz NOT NULL DEFAULT now(),
  ends_at timestamptz,
  city_scope jsonb NOT NULL DEFAULT '[]'::jsonb,
  user_scope jsonb NOT NULL DEFAULT '[]'::jsonb,
  total_reward_limit integer,
  daily_reward_limit integer,
  config_snapshot jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_by bigint REFERENCES admin_operator_profiles(id),
  updated_by bigint REFERENCES admin_operator_profiles(id),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CHECK (status IN ('draft', 'active', 'paused', 'ended', 'disabled')),
  CHECK (ends_at IS NULL OR ends_at > starts_at),
  CHECK (total_reward_limit IS NULL OR total_reward_limit > 0),
  CHECK (daily_reward_limit IS NULL OR daily_reward_limit > 0)
);

CREATE TABLE IF NOT EXISTS growth_campaign_rules (
  id bigint PRIMARY KEY DEFAULT next_tsid(),
  campaign_code varchar(64) NOT NULL REFERENCES growth_campaigns(code),
  rule_code varchar(64) NOT NULL,
  trigger_event varchar(64) NOT NULL,
  status varchar(32) NOT NULL DEFAULT 'active',
  priority integer NOT NULL DEFAULT 100,
  conditions jsonb NOT NULL DEFAULT '{}'::jsonb,
  reward_type varchar(64) NOT NULL,
  reward_amount integer NOT NULL,
  valid_days integer NOT NULL,
  per_user_limit integer,
  per_user_daily_limit integer,
  per_resource_daily_limit integer,
  description varchar(255) NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (campaign_code, rule_code),
  CHECK (status IN ('active', 'inactive')),
  CHECK (trigger_event IN ('user_first_login', 'resource_first_approved', 'resource_approved_count_reached', 'resource_share_effective_view', 'resource_share_effective_contact', 'invitee_first_resource_approved')),
  CHECK (reward_type IN ('publish_quota', 'refresh_quota', 'top_voucher')),
  CHECK (reward_amount > 0),
  CHECK (valid_days > 0),
  CHECK (per_user_limit IS NULL OR per_user_limit > 0),
  CHECK (per_user_daily_limit IS NULL OR per_user_daily_limit > 0),
  CHECK (per_resource_daily_limit IS NULL OR per_resource_daily_limit > 0)
);

CREATE TABLE IF NOT EXISTS growth_reward_grants (
  id bigint PRIMARY KEY DEFAULT next_tsid(),
  campaign_code varchar(64) NOT NULL,
  rule_code varchar(64) NOT NULL,
  merchant_id bigint NOT NULL REFERENCES merchants(id),
  user_id bigint REFERENCES users(id),
  resource_id bigint REFERENCES resources(id),
  event_id bigint,
  idempotency_key varchar(255) NOT NULL,
  reward_type varchar(64) NOT NULL,
  reward_amount integer NOT NULL,
  valid_days integer NOT NULL,
  entitlement_id bigint REFERENCES merchant_entitlements(id),
  status varchar(32) NOT NULL DEFAULT 'granted',
  reason varchar(128) NOT NULL DEFAULT '',
  snapshot jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (idempotency_key),
  CHECK (status IN ('granted', 'skipped', 'failed', 'revoked')),
  CHECK (reward_amount > 0),
  CHECK (valid_days > 0)
);

CREATE INDEX IF NOT EXISTS idx_growth_campaigns_status_time
  ON growth_campaigns(status, starts_at, ends_at);
CREATE INDEX IF NOT EXISTS idx_growth_campaign_rules_campaign_status
  ON growth_campaign_rules(campaign_code, status, trigger_event, priority);
CREATE INDEX IF NOT EXISTS idx_growth_reward_grants_merchant
  ON growth_reward_grants(merchant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_growth_reward_grants_resource
  ON growth_reward_grants(resource_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_growth_reward_grants_campaign_rule
  ON growth_reward_grants(campaign_code, rule_code, created_at DESC);

INSERT INTO growth_campaigns (
  code, name, status, starts_at, ends_at, config_snapshot
) VALUES (
  'starter_growth_2026_q3',
  '新手成长权益活动',
  'paused',
  now(),
  now() + interval '90 days',
  '{"frontendTitle":"新手发布权益","frontendHint":"发布优质资源、有效分享可获得更多曝光权益"}'::jsonb
) ON CONFLICT (code) DO NOTHING;

INSERT INTO growth_campaign_rules (
  campaign_code, rule_code, trigger_event, status, priority, conditions, reward_type, reward_amount, valid_days, per_user_limit, per_user_daily_limit, per_resource_daily_limit, description
) VALUES
  ('starter_growth_2026_q3', 'first_login_publish_quota', 'user_first_login', 'active', 10, '{}'::jsonb, 'publish_quota', 3, 30, 1, NULL, NULL, '首次登录赠送发布次数'),
  ('starter_growth_2026_q3', 'first_resource_approved_publish_quota', 'resource_first_approved', 'active', 20, '{}'::jsonb, 'publish_quota', 5, 30, 1, NULL, NULL, '首条资源审核通过赠送发布次数'),
  ('starter_growth_2026_q3', 'three_approved_resources_publish_quota', 'resource_approved_count_reached', 'active', 30, '{"approvedCount":3,"windowDays":30}'::jsonb, 'publish_quota', 5, 30, 1, NULL, NULL, '连续优质发布赠送发布次数'),
  ('starter_growth_2026_q3', 'three_approved_resources_refresh_quota', 'resource_approved_count_reached', 'active', 31, '{"approvedCount":3,"windowDays":30}'::jsonb, 'refresh_quota', 3, 30, 1, NULL, NULL, '连续优质发布赠送刷新次数'),
  ('starter_growth_2026_q3', 'share_contact_refresh_quota', 'resource_share_effective_contact', 'active', 40, '{}'::jsonb, 'refresh_quota', 1, 15, NULL, 3, NULL, '分享带来有效联系赠送刷新次数'),
  ('starter_growth_2026_q3', 'share_view_refresh_quota', 'resource_share_effective_view', 'inactive', 50, '{"viewThreshold":5,"windowHours":24}'::jsonb, 'refresh_quota', 1, 15, NULL, 3, 1, '分享有效浏览奖励，首期默认关闭'),
  ('starter_growth_2026_q3', 'invitee_first_resource_approved', 'invitee_first_resource_approved', 'inactive', 60, '{}'::jsonb, 'publish_quota', 5, 30, NULL, NULL, NULL, '邀请奖励预留，等待邀请关系模型')
ON CONFLICT (campaign_code, rule_code) DO NOTHING;
```

Create `backend/migrations/000020_growth_campaigns.down.sql`:

```sql
DROP INDEX IF EXISTS idx_growth_reward_grants_campaign_rule;
DROP INDEX IF EXISTS idx_growth_reward_grants_resource;
DROP INDEX IF EXISTS idx_growth_reward_grants_merchant;
DROP INDEX IF EXISTS idx_growth_campaign_rules_campaign_status;
DROP INDEX IF EXISTS idx_growth_campaigns_status_time;
DROP TABLE IF EXISTS growth_reward_grants;
DROP TABLE IF EXISTS growth_campaign_rules;
DROP TABLE IF EXISTS growth_campaigns;
```

- [ ] **Step 4: Run migration test**

Run:

```bash
node --test backend/scripts/validate_migrations.test.mjs
```

Expected: PASS.

## Task 2: 增长活动模型与幂等发放

**Files:**
- Create: `backend/app/internal/model/growth_campaign_model.go`
- Create: `backend/app/internal/model/growth_campaign_model_test.go`
- Modify: `backend/app/internal/svc/service_context.go`

- [ ] **Step 1: Write failing model tests**

Create `backend/app/internal/model/growth_campaign_model_test.go`:

```go
package model

import (
	"os"
	"strings"
	"testing"
)

func TestGrowthCampaignModelUsesCampaignSourceAndIdempotency(t *testing.T) {
	source, err := os.ReadFile("growth_campaign_model.go")
	if err != nil {
		t.Fatalf("ReadFile(growth_campaign_model.go) error = %v", err)
	}
	text := string(source)

	for _, snippet := range []string{
		`EntitlementSourceGrowthCampaign = "growth_campaign"`,
		"growth_campaigns",
		"growth_campaign_rules",
		"growth_reward_grants",
		"idempotency_key",
		"ON CONFLICT (idempotency_key) DO NOTHING",
		"INSERT INTO merchant_entitlements",
		"source_type",
		"expires_at",
	} {
		if !strings.Contains(text, snippet) {
			t.Fatalf("growth campaign model missing snippet %q", snippet)
		}
	}
}

func TestGrowthCampaignModelIgnoresInactiveCampaignStatuses(t *testing.T) {
	source, err := os.ReadFile("growth_campaign_model.go")
	if err != nil {
		t.Fatalf("ReadFile(growth_campaign_model.go) error = %v", err)
	}
	text := string(source)

	for _, snippet := range []string{
		"gc.status = 'active'",
		"gcr.status = 'active'",
		"gc.starts_at <= now()",
		"(gc.ends_at IS NULL OR gc.ends_at > now())",
	} {
		if !strings.Contains(text, snippet) {
			t.Fatalf("growth campaign active query missing snippet %q", snippet)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
cd backend/app/internal/model && go test ./... -run GrowthCampaign
```

Expected: FAIL because `growth_campaign_model.go` does not exist.

- [ ] **Step 3: Add model implementation**

Create `backend/app/internal/model/growth_campaign_model.go`:

```go
package model

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

const (
	EntitlementSourceGrowthCampaign = "growth_campaign"

	GrowthEventUserFirstLogin              = "user_first_login"
	GrowthEventResourceFirstApproved       = "resource_first_approved"
	GrowthEventResourceApprovedCountReached = "resource_approved_count_reached"
	GrowthEventResourceShareEffectiveView  = "resource_share_effective_view"
	GrowthEventResourceShareEffectiveContact = "resource_share_effective_contact"
)

type GrowthEventInput struct {
	EventType  string
	MerchantID string
	UserID     string
	ResourceID string
	EventID    string
	OccurredAt time.Time
}

type GrowthRewardGrantResult struct {
	ID            string
	EntitlementID string
	Status        string
	Reason        string
}

type GrowthCampaignModel struct {
	db *sql.DB
}

func NewGrowthCampaignModel(db *sql.DB) *GrowthCampaignModel {
	return &GrowthCampaignModel{db: db}
}

func (m *GrowthCampaignModel) TriggerGrowthEvent(ctx context.Context, input GrowthEventInput) ([]GrowthRewardGrantResult, error) {
	input.EventType = strings.TrimSpace(input.EventType)
	input.MerchantID = strings.TrimSpace(input.MerchantID)
	input.UserID = strings.TrimSpace(input.UserID)
	input.ResourceID = strings.TrimSpace(input.ResourceID)
	input.EventID = strings.TrimSpace(input.EventID)
	if input.EventType == "" {
		return nil, nil
	}
	if input.OccurredAt.IsZero() {
		input.OccurredAt = time.Now().UTC()
	}
	if input.MerchantID == "" && input.ResourceID != "" {
		merchantID, err := m.resourceMerchantID(ctx, input.ResourceID)
		if err != nil {
			return nil, err
		}
		input.MerchantID = merchantID
	}
	if input.MerchantID == "" {
		return nil, nil
	}

	rules, err := m.listActiveGrowthRules(ctx, input.EventType)
	if err != nil {
		return nil, err
	}
	results := make([]GrowthRewardGrantResult, 0, len(rules))
	for _, rule := range rules {
		result, err := m.grantGrowthRule(ctx, input, rule)
		if err != nil {
			logx.Errorf("成长权益发放失败: campaignCode=%s ruleCode=%s eventType=%s merchantId=%s resourceId=%s eventId=%s err=%+v", rule.CampaignCode, rule.RuleCode, input.EventType, input.MerchantID, input.ResourceID, input.EventID, err)
			return results, err
		}
		if result.ID != "" {
			results = append(results, result)
		}
	}
	return results, nil
}

type activeGrowthRule struct {
	CampaignCode string
	RuleCode     string
	RewardType   string
	RewardAmount int64
	ValidDays    int64
}

func (m *GrowthCampaignModel) listActiveGrowthRules(ctx context.Context, eventType string) ([]activeGrowthRule, error) {
	rows, err := m.db.QueryContext(ctx, `
SELECT gcr.campaign_code, gcr.rule_code, gcr.reward_type, gcr.reward_amount, gcr.valid_days
FROM growth_campaign_rules gcr
JOIN growth_campaigns gc ON gc.code = gcr.campaign_code
WHERE gc.status = 'active'
  AND gcr.status = 'active'
  AND gcr.trigger_event = $1
  AND gc.starts_at <= now()
  AND (gc.ends_at IS NULL OR gc.ends_at > now())
ORDER BY gcr.priority ASC, gcr.id ASC
`, eventType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []activeGrowthRule
	for rows.Next() {
		var rule activeGrowthRule
		if err := rows.Scan(&rule.CampaignCode, &rule.RuleCode, &rule.RewardType, &rule.RewardAmount, &rule.ValidDays); err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}
	return rules, rows.Err()
}

func (m *GrowthCampaignModel) grantGrowthRule(ctx context.Context, input GrowthEventInput, rule activeGrowthRule) (GrowthRewardGrantResult, error) {
	var result GrowthRewardGrantResult
	idempotencyKey := growthRewardIdempotencyKey(input, rule)
	expiresAt := input.OccurredAt.AddDate(0, 0, int(rule.ValidDays))
	err := WithTx(ctx, m.db, func(tx *sql.Tx) error {
		var entitlementID string
		err := tx.QueryRowContext(ctx, `
INSERT INTO growth_reward_grants (
  campaign_code, rule_code, merchant_id, user_id, resource_id, event_id, idempotency_key,
  reward_type, reward_amount, valid_days, status, reason, snapshot
) VALUES (
  $1, $2, $3, NULLIF($4, '')::bigint, NULLIF($5, '')::bigint, NULLIF($6, '')::bigint, $7,
  $8, $9, $10, 'granted', 'matched', '{}'::jsonb
)
ON CONFLICT (idempotency_key) DO NOTHING
RETURNING id::text, status, reason
`, rule.CampaignCode, rule.RuleCode, input.MerchantID, input.UserID, input.ResourceID, input.EventID, idempotencyKey, rule.RewardType, rule.RewardAmount, rule.ValidDays).Scan(&result.ID, &result.Status, &result.Reason)
		if err == sql.ErrNoRows {
			result.Status = "skipped"
			result.Reason = "duplicate"
			return nil
		}
		if err != nil {
			return err
		}
		if err := tx.QueryRowContext(ctx, `
INSERT INTO merchant_entitlements (
  merchant_id, entitlement_type, source_type, total_amount, remaining_amount, starts_at, expires_at, status
) VALUES (
  $1, $2, $3, $4, $4, now(), $5, 'active'
) RETURNING id::text
`, input.MerchantID, rule.RewardType, EntitlementSourceGrowthCampaign, rule.RewardAmount, expiresAt).Scan(&entitlementID); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `
UPDATE growth_reward_grants
SET entitlement_id = $2
WHERE id = $1
`, result.ID, entitlementID)
		if err == nil {
			result.EntitlementID = entitlementID
		}
		return err
	})
	return result, err
}

func (m *GrowthCampaignModel) resourceMerchantID(ctx context.Context, resourceID string) (string, error) {
	var merchantID string
	err := m.db.QueryRowContext(ctx, `
SELECT merchant_id::text
FROM resources
WHERE id = $1
  AND deleted_at IS NULL
`, resourceID).Scan(&merchantID)
	return merchantID, err
}

func growthRewardIdempotencyKey(input GrowthEventInput, rule activeGrowthRule) string {
	objectKey := input.EventID
	if objectKey == "" {
		objectKey = input.ResourceID
	}
	if objectKey == "" {
		objectKey = input.MerchantID
	}
	return fmt.Sprintf("%s:%s:%s:%s", rule.CampaignCode, rule.RuleCode, input.MerchantID, objectKey)
}
```

- [ ] **Step 4: Wire model into APIStore**

Modify `backend/app/internal/svc/service_context.go`:

```go
*model.OperationLogModel
*model.GrowthCampaignModel
*model.FavoriteModel
```

and inside `newAPIStore`:

```go
OperationLogModel:         model.NewOperationLogModel(db),
GrowthCampaignModel: model.NewGrowthCampaignModel(db),
FavoriteModel:             model.NewFavoriteModel(db),
```

- [ ] **Step 5: Run model tests**

Run:

```bash
cd backend/app/internal/model && go test ./...
```

Expected: PASS.

## Task 3: 统一增长事件逻辑

**Files:**
- Create: `backend/app/internal/logic/growth/growth_reward_logic.go`
- Create: `backend/app/internal/logic/growth/growth_reward_logic_test.go`

- [ ] **Step 1: Write failing logic tests**

Create `backend/app/internal/logic/growth/growth_reward_logic_test.go`:

```go
package growth

import (
	"context"
	"errors"
	"testing"

	"wplink/backend/app/internal/model"
)

func TestProcessGrowthEventTrimsAndPassesInput(t *testing.T) {
	store := &fakeGrowthStore{}
	logic := NewRewardLogic(store)

	err := logic.ProcessEvent(context.Background(), model.GrowthEventInput{
		EventType: " user_first_login ",
		MerchantID: " merchant-1 ",
		UserID: " user-1 ",
	})
	if err != nil {
		t.Fatalf("ProcessEvent() error = %v", err)
	}
	if store.input.EventType != "user_first_login" || store.input.MerchantID != "merchant-1" || store.input.UserID != "user-1" {
		t.Fatalf("input = %#v, want trimmed growth event", store.input)
	}
}

func TestProcessGrowthEventSkipsMissingStore(t *testing.T) {
	logic := NewRewardLogic(nil)
	if err := logic.ProcessEvent(context.Background(), model.GrowthEventInput{EventType: "user_first_login"}); err != nil {
		t.Fatalf("ProcessEvent() error = %v, want nil when store missing", err)
	}
}

func TestProcessGrowthEventReturnsStoreError(t *testing.T) {
	logic := NewRewardLogic(&fakeGrowthStore{err: errors.New("db down")})
	err := logic.ProcessEvent(context.Background(), model.GrowthEventInput{EventType: "user_first_login", MerchantID: "merchant-1"})
	if err == nil {
		t.Fatal("ProcessEvent() error = nil, want store error")
	}
}

type fakeGrowthStore struct {
	input model.GrowthEventInput
	err   error
}

func (s *fakeGrowthStore) TriggerGrowthEvent(ctx context.Context, input model.GrowthEventInput) ([]model.GrowthRewardGrantResult, error) {
	s.input = input
	return nil, s.err
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
cd backend/app/internal/logic/growth && go test ./...
```

Expected: FAIL because package files do not exist.

- [ ] **Step 3: Add growth logic**

Create `backend/app/internal/logic/growth/growth_reward_logic.go`:

```go
package growth

import (
	"context"
	"strings"

	"wplink/backend/app/internal/model"
)

type RewardStore interface {
	TriggerGrowthEvent(ctx context.Context, input model.GrowthEventInput) ([]model.GrowthRewardGrantResult, error)
}

type RewardLogic struct {
	store RewardStore
}

func NewRewardLogic(store RewardStore) *RewardLogic {
	return &RewardLogic{store: store}
}

func (l *RewardLogic) ProcessEvent(ctx context.Context, input model.GrowthEventInput) error {
	if l == nil || l.store == nil {
		return nil
	}
	input.EventType = strings.TrimSpace(input.EventType)
	input.MerchantID = strings.TrimSpace(input.MerchantID)
	input.UserID = strings.TrimSpace(input.UserID)
	input.ResourceID = strings.TrimSpace(input.ResourceID)
	input.EventID = strings.TrimSpace(input.EventID)
	if input.EventType == "" {
		return nil
	}
	_, err := l.store.TriggerGrowthEvent(ctx, input)
	return err
}
```

- [ ] **Step 4: Run logic tests**

Run:

```bash
cd backend/app/internal/logic/growth && go test ./...
```

Expected: PASS.

## Task 4: 接入登录、审核和联系事件

**Files:**
- Modify: `backend/app/internal/logic/auth/auth_logic.go`
- Modify: `backend/app/internal/logic/auth/auth_logic_test.go`
- Modify: `backend/app/internal/logic/admin/review_resource_logic.go`
- Modify: `backend/app/internal/logic/admin/review_resource_logic_test.go`
- Modify: `backend/app/internal/logic/metrics/record_contact_logic.go`
- Modify: `backend/app/internal/logic/metrics/record_contact_logic_test.go`

- [ ] **Step 1: Write failing tests for login trigger**

Add to `backend/app/internal/logic/auth/auth_logic_test.go`:

```go
func TestWechatLoginTriggersGrowthRewardAfterDefaultMerchantReady(t *testing.T) {
	tokenService := &fakeTokenService{}
	sessionClient := &fakeWechatSessionClient{session: WechatSession{OpenID: "openid-1"}}
	store := &fakeUserStore{
		profile: model.UserProfile{ID: "user-1"},
		defaultMerchant: model.ManagedMerchantInfo{ID: "merchant-1", ProfileStatus: model.MerchantProfileStatusIncomplete},
	}
	logic := NewWechatLoginLogic(store, tokenService, sessionClient)

	_, err := logic.WechatLogin(context.Background(), WechatLoginReq{Code: "wx-code", DefaultCityCode: "zhili"})
	if err != nil {
		t.Fatalf("WechatLogin() error = %v", err)
	}
	if store.growthInput.EventType != model.GrowthEventUserFirstLogin || store.growthInput.MerchantID != "merchant-1" || store.growthInput.UserID != "user-1" {
		t.Fatalf("growthInput = %#v, want first login event", store.growthInput)
	}
}
```

Extend the fake store in the same test file:

```go
growthInput model.GrowthEventInput

func (s *fakeUserStore) TriggerGrowthEvent(ctx context.Context, input model.GrowthEventInput) ([]model.GrowthRewardGrantResult, error) {
	s.growthInput = input
	return nil, nil
}
```

- [ ] **Step 2: Add login hook**

In `backend/app/internal/logic/auth/auth_logic.go`, define a local optional interface:

```go
type GrowthEventStore interface {
	TriggerGrowthEvent(ctx context.Context, input model.GrowthEventInput) ([]model.GrowthRewardGrantResult, error)
}
```

After `profile, err = l.ensureDefaultMerchantProfile(...)` succeeds:

```go
if growthStore, ok := l.store.(GrowthEventStore); ok {
	for _, merchant := range profile.ManagedMerchants {
		if strings.TrimSpace(merchant.ID) == "" {
			continue
		}
		if _, err := growthStore.TriggerGrowthEvent(ctx, model.GrowthEventInput{
			EventType:  model.GrowthEventUserFirstLogin,
			MerchantID: merchant.ID,
			UserID:     profile.ID,
		}); err != nil {
			logx.Errorf("登录后触发新手成长权益失败: userId=%s merchantId=%s err=%+v", profile.ID, merchant.ID, err)
		}
		break
	}
}
```

- [ ] **Step 3: Write failing tests for review trigger**

Add to `backend/app/internal/logic/admin/review_resource_logic_test.go`:

```go
func TestReviewResourceTriggersGrowthRewardOnlyWhenApproved(t *testing.T) {
	store := &fakeReviewResourceStore{result: model.ReviewResourceResult{ID: "resource-1", Status: "published"}}
	logic := NewReviewResourceLogic(store)

	_, err := logic.ReviewResource(context.Background(), "resource-1", ReviewResourceReq{Action: "approve", ReviewerID: "admin-1"})
	if err != nil {
		t.Fatalf("ReviewResource() error = %v", err)
	}
	if store.growthInputs[0].EventType != model.GrowthEventResourceFirstApproved || store.growthInputs[0].ResourceID != "resource-1" {
		t.Fatalf("growthInputs = %#v, want first approved event", store.growthInputs)
	}
	if store.growthInputs[1].EventType != model.GrowthEventResourceApprovedCountReached {
		t.Fatalf("growthInputs = %#v, want approved count event", store.growthInputs)
	}
}
```

Extend the fake store:

```go
growthInputs []model.GrowthEventInput

func (s *fakeReviewResourceStore) TriggerGrowthEvent(ctx context.Context, input model.GrowthEventInput) ([]model.GrowthRewardGrantResult, error) {
	s.growthInputs = append(s.growthInputs, input)
	return nil, nil
}
```

- [ ] **Step 4: Add review hook**

In `backend/app/internal/logic/admin/review_resource_logic.go`, after success log and before return:

```go
if action == "approve" && result.Status == model.ResourceStatusPublished {
	if growthStore, ok := l.store.(interface {
		TriggerGrowthEvent(context.Context, model.GrowthEventInput) ([]model.GrowthRewardGrantResult, error)
	}); ok {
		for _, eventType := range []string{model.GrowthEventResourceFirstApproved, model.GrowthEventResourceApprovedCountReached} {
			if _, err := growthStore.TriggerGrowthEvent(ctx, model.GrowthEventInput{EventType: eventType, ResourceID: result.ID}); err != nil {
				logx.Errorf("资源审核通过后触发成长权益失败: resourceId=%s reviewerId=%s eventType=%s err=%+v", result.ID, strings.TrimSpace(req.ReviewerID), eventType, err)
			}
		}
	}
}
```

- [ ] **Step 5: Write failing tests for contact trigger**

Add to `backend/app/internal/logic/metrics/record_contact_logic_test.go`:

```go
func TestRecordContactTriggersGrowthRewardForEffectiveContact(t *testing.T) {
	store := &fakeContactStore{
		unlockInfo: model.ResourceContactUnlockInfo{ResourceID: "resource-1", MerchantID: "merchant-1", Status: model.ResourceStatusPublished, Phone: "18800000001"},
		contactResult: model.ResourceContactEventResult{ID: "event-1", MerchantID: "merchant-1"},
	}
	logic := NewRecordContactLogic(store)

	_, err := logic.RecordContact(context.Background(), RecordContactReq{ResourceID: "resource-1", UserID: "user-2", Action: "phone"})
	if err != nil {
		t.Fatalf("RecordContact() error = %v", err)
	}
	if store.growthInput.EventType != model.GrowthEventResourceShareEffectiveContact || store.growthInput.EventID != "event-1" {
		t.Fatalf("growthInput = %#v, want share contact event", store.growthInput)
	}
}
```

Extend the fake store:

```go
growthInput model.GrowthEventInput

func (s *fakeContactStore) TriggerGrowthEvent(ctx context.Context, input model.GrowthEventInput) ([]model.GrowthRewardGrantResult, error) {
	s.growthInput = input
	return nil, nil
}
```

- [ ] **Step 6: Add contact hook and share_view action**

In `backend/app/internal/logic/metrics/record_contact_logic.go`:

```go
func isSupportedContactAction(action string) bool {
	switch action {
	case "phone", "wechat", "merchant_home", "merchant_profile", "share", "share_view":
		return true
	default:
		return false
	}
}
```

After `RecordResourceContactEvent` succeeds, keep the result and trigger growth:

```go
eventResult, err := l.store.RecordResourceContactEvent(ctx, input)
if err != nil {
	return RecordContactResp{}, err
}
if growthStore, ok := l.store.(interface {
	TriggerGrowthEvent(context.Context, model.GrowthEventInput) ([]model.GrowthRewardGrantResult, error)
}); ok {
	eventType := ""
	if input.Action == "phone" || input.Action == "wechat" {
		eventType = model.GrowthEventResourceShareEffectiveContact
	}
	if input.Action == "share_view" {
		eventType = model.GrowthEventResourceShareEffectiveView
	}
	if eventType != "" {
		if _, err := growthStore.TriggerGrowthEvent(ctx, model.GrowthEventInput{
			EventType:  eventType,
			MerchantID: eventResult.MerchantID,
			UserID:     input.UserID,
			ResourceID: input.ResourceID,
			EventID:    eventResult.ID,
		}); err != nil {
			logx.Errorf("联系事件触发成长权益失败: resourceId=%s userId=%s action=%s eventId=%s err=%+v", input.ResourceID, input.UserID, input.Action, eventResult.ID, err)
		}
	}
}
```

- [ ] **Step 7: Run logic tests**

Run:

```bash
cd backend/app/internal/logic && go test ./...
```

Expected: PASS.

## Task 5: 弱化商家名片对应的发布额度改造

**Files:**
- Modify: `backend/app/internal/model/resource_model.go`
- Modify: `backend/app/internal/model/resource_model_test.go`

- [ ] **Step 1: Write failing tests**

Update the existing profile monthly quota tests in `backend/app/internal/model/resource_model_test.go` so completed profile no longer receives more quota than incomplete profile:

```go
func TestProfileMonthlyBenefitsNoLongerRewardMerchantCardCompletion(t *testing.T) {
	incompletePublishQuota, incompleteRefreshQuota := profileMonthlyBenefitsForStatus(MerchantProfileStatusIncomplete)
	completedPublishQuota, completedRefreshQuota := profileMonthlyBenefitsForStatus(MerchantProfileStatusCompleted)

	if incompletePublishQuota != 0 || incompleteRefreshQuota != 0 {
		t.Fatalf("incomplete profile quota = %d/%d, want 0/0 after growth campaign migration", incompletePublishQuota, incompleteRefreshQuota)
	}
	if completedPublishQuota != 0 || completedRefreshQuota != 0 {
		t.Fatalf("completed profile quota = %d/%d, want 0/0 to avoid merchant card incentive", completedPublishQuota, completedRefreshQuota)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
cd backend/app/internal/model && go test ./... -run ProfileMonthlyBenefits
```

Expected: FAIL because current completed profile still returns `10, 3`.

- [ ] **Step 3: Change profile monthly benefits**

Modify `profileMonthlyBenefitsForStatus` in `backend/app/internal/model/resource_model.go`:

```go
func profileMonthlyBenefitsForStatus(profileStatus string) (int64, int64) {
	// 普通用户冷启动发布次数改由 growth_campaign 发放，避免继续把“完善商家名片”作为核心激励。
	return 0, 0
}
```

- [ ] **Step 4: Run model tests**

Run:

```bash
cd backend/app/internal/model && go test ./...
```

Expected: PASS.

## Task 6: 后台增长活动接口

**Files:**
- Create: `backend/app/internal/logic/admin/growth_campaign_logic.go`
- Create: `backend/app/internal/logic/admin/growth_campaign_logic_test.go`
- Modify: `backend/app/internal/server/domain_routes.go`
- Modify: `backend/app/internal/server/remaining_api_test.go`

- [ ] **Step 1: Write failing admin logic tests**

Create `backend/app/internal/logic/admin/growth_campaign_logic_test.go`:

```go
package admin

import (
	"context"
	"testing"

	"wplink/backend/common/errx"
)

func TestSaveGrowthRuleRejectsShareRewardWithoutLimit(t *testing.T) {
	logic := NewGrowthCampaignLogic(&fakeGrowthCampaignAdminStore{})
	_, err := logic.SaveGrowthRule(context.Background(), SaveGrowthRuleReq{
		CampaignCode: "starter_growth_2026_q3",
		RuleCode: "share_contact_refresh_quota",
		TriggerEvent: "resource_share_effective_contact",
		Status: "active",
		RewardType: "refresh_quota",
		RewardAmount: 1,
		ValidDays: 15,
	})
	if err == nil || errx.CodeOf(err) != errx.CodeValidationFailed {
		t.Fatalf("SaveGrowthRule() error = %v, want validation error", err)
	}
}

func TestDisableGrowthCampaignRequiresReason(t *testing.T) {
	logic := NewGrowthCampaignLogic(&fakeGrowthCampaignAdminStore{})
	_, err := logic.SaveGrowthCampaign(context.Background(), SaveGrowthCampaignReq{
		Code: "starter_growth_2026_q3",
		Name: "新手成长权益活动",
		Status: "disabled",
	})
	if err == nil || errx.CodeOf(err) != errx.CodeValidationFailed {
		t.Fatalf("SaveGrowthCampaign() error = %v, want validation error", err)
	}
}
```

- [ ] **Step 2: Add admin logic**

Create `backend/app/internal/logic/admin/growth_campaign_logic.go` with request/response structs and validation:

```go
package admin

import (
	"context"
	"strings"

	"wplink/backend/common/errx"
)

type GrowthCampaignAdminStore interface{}

type SaveGrowthCampaignReq struct {
	Code string
	Name string
	Status string
	DisableReason string
	OperatorID string
}

type SaveGrowthCampaignResp struct {
	Code string `json:"code"`
	Message string `json:"message"`
}

type ListGrowthCampaignsReq struct {
	Status string
}

type GrowthCampaignItem struct {
	Code string `json:"code"`
	Name string `json:"name"`
	Status string `json:"status"`
	StartsAt string `json:"startsAt,omitempty"`
	EndsAt string `json:"endsAt,omitempty"`
}

type ListGrowthCampaignsResp struct {
	Items []GrowthCampaignItem `json:"items"`
}

type SaveGrowthRuleReq struct {
	CampaignCode string
	RuleCode string
	TriggerEvent string
	Status string
	RewardType string
	RewardAmount int64
	ValidDays int64
	PerUserDailyLimit int64
	PerResourceDailyLimit int64
	OperatorID string
}

type SaveGrowthRuleResp struct {
	RuleCode string `json:"ruleCode"`
	Message string `json:"message"`
}

type ListGrowthRewardGrantsReq struct {
	CampaignCode string
	Status string
	MerchantID string
}

type GrowthRewardGrantItem struct {
	ID string `json:"id"`
	CampaignCode string `json:"campaignCode"`
	RuleCode string `json:"ruleCode"`
	MerchantID string `json:"merchantId"`
	ResourceID string `json:"resourceId,omitempty"`
	RewardType string `json:"rewardType"`
	RewardAmount int64 `json:"rewardAmount"`
	Status string `json:"status"`
	CreatedAt string `json:"createdAt"`
}

type ListGrowthRewardGrantsResp struct {
	Items []GrowthRewardGrantItem `json:"items"`
}

type GrowthCampaignLogic struct {
	store GrowthCampaignAdminStore
}

func NewGrowthCampaignLogic(store GrowthCampaignAdminStore) *GrowthCampaignLogic {
	return &GrowthCampaignLogic{store: store}
}

func (l *GrowthCampaignLogic) ListGrowthCampaigns(ctx context.Context, req ListGrowthCampaignsReq) (ListGrowthCampaignsResp, error) {
	if strings.TrimSpace(req.Status) != "" && !isGrowthCampaignStatus(strings.TrimSpace(req.Status)) {
		return ListGrowthCampaignsResp{}, errx.New(errx.CodeValidationFailed, "活动状态不正确")
	}
	return ListGrowthCampaignsResp{Items: []GrowthCampaignItem{}}, nil
}

func (l *GrowthCampaignLogic) SaveGrowthCampaign(ctx context.Context, req SaveGrowthCampaignReq) (SaveGrowthCampaignResp, error) {
	code := strings.TrimSpace(req.Code)
	name := strings.TrimSpace(req.Name)
	status := strings.TrimSpace(req.Status)
	if code == "" || name == "" {
		return SaveGrowthCampaignResp{}, errx.New(errx.CodeValidationFailed, "请填写活动编码和名称")
	}
	if !isGrowthCampaignStatus(status) {
		return SaveGrowthCampaignResp{}, errx.New(errx.CodeValidationFailed, "活动状态不正确")
	}
	if status == "disabled" && strings.TrimSpace(req.DisableReason) == "" {
		return SaveGrowthCampaignResp{}, errx.New(errx.CodeValidationFailed, "停用活动必须填写原因")
	}
	return SaveGrowthCampaignResp{Code: code, Message: "增长活动已保存"}, nil
}

func (l *GrowthCampaignLogic) SaveGrowthRule(ctx context.Context, req SaveGrowthRuleReq) (SaveGrowthRuleResp, error) {
	ruleCode := strings.TrimSpace(req.RuleCode)
	if strings.TrimSpace(req.CampaignCode) == "" || ruleCode == "" {
		return SaveGrowthRuleResp{}, errx.New(errx.CodeValidationFailed, "请填写活动和规则编码")
	}
	if req.RewardAmount <= 0 {
		return SaveGrowthRuleResp{}, errx.New(errx.CodeValidationFailed, "奖励数量必须大于 0")
	}
	if req.ValidDays <= 0 {
		return SaveGrowthRuleResp{}, errx.New(errx.CodeValidationFailed, "权益有效期必须大于 0")
	}
	if strings.Contains(strings.TrimSpace(req.TriggerEvent), "share") && req.PerUserDailyLimit <= 0 && req.PerResourceDailyLimit <= 0 {
		return SaveGrowthRuleResp{}, errx.New(errx.CodeValidationFailed, "分享类奖励必须设置每日或资源上限")
	}
	return SaveGrowthRuleResp{RuleCode: ruleCode, Message: "增长规则已保存"}, nil
}

func (l *GrowthCampaignLogic) ListGrowthRewardGrants(ctx context.Context, req ListGrowthRewardGrantsReq) (ListGrowthRewardGrantsResp, error) {
	if strings.TrimSpace(req.CampaignCode) == "" {
		return ListGrowthRewardGrantsResp{}, errx.New(errx.CodeValidationFailed, "活动不存在")
	}
	return ListGrowthRewardGrantsResp{Items: []GrowthRewardGrantItem{}}, nil
}

func isGrowthCampaignStatus(status string) bool {
	switch status {
	case "draft", "active", "paused", "ended", "disabled":
		return true
	default:
		return false
	}
}
```

- [ ] **Step 3: Add routes**

In `backend/app/internal/server/domain_routes.go`, register:

```go
mux.HandleFunc("GET /api/v1/admin/growth-campaigns", func(w http.ResponseWriter, r *http.Request) {
	resp, err := adminlogic.NewGrowthCampaignLogic(store).ListGrowthCampaigns(r.Context(), adminlogic.ListGrowthCampaignsReq{
		Status: r.URL.Query().Get("status"),
	})
	response.JSON(w, resp, err)
})
mux.HandleFunc("POST /api/v1/admin/growth-campaigns", func(w http.ResponseWriter, r *http.Request) {
	var body adminlogic.SaveGrowthCampaignReq
	if err := decodeJSONBody(r, &body); err != nil {
		response.JSON(w, nil, err)
		return
	}
	operatorID, err := adminOperatorIDFromRequest(r, adminTokenService, body.OperatorID)
	if err != nil {
		response.JSON(w, nil, err)
		return
	}
	body.OperatorID = operatorID
	resp, err := adminlogic.NewGrowthCampaignLogic(store).SaveGrowthCampaign(r.Context(), body)
	response.JSON(w, resp, err)
})
mux.HandleFunc("POST /api/v1/admin/growth-campaigns/{campaignCode}/rules", func(w http.ResponseWriter, r *http.Request) {
	var body adminlogic.SaveGrowthRuleReq
	if err := decodeJSONBody(r, &body); err != nil {
		response.JSON(w, nil, err)
		return
	}
	body.CampaignCode = r.PathValue("campaignCode")
	operatorID, err := adminOperatorIDFromRequest(r, adminTokenService, body.OperatorID)
	if err != nil {
		response.JSON(w, nil, err)
		return
	}
	body.OperatorID = operatorID
	resp, err := adminlogic.NewGrowthCampaignLogic(store).SaveGrowthRule(r.Context(), body)
	response.JSON(w, resp, err)
})
mux.HandleFunc("GET /api/v1/admin/growth-campaigns/{campaignCode}/grants", func(w http.ResponseWriter, r *http.Request) {
	resp, err := adminlogic.NewGrowthCampaignLogic(store).ListGrowthRewardGrants(r.Context(), adminlogic.ListGrowthRewardGrantsReq{
		CampaignCode: r.PathValue("campaignCode"),
		Status:       r.URL.Query().Get("status"),
		MerchantID:   r.URL.Query().Get("merchantId"),
	})
	response.JSON(w, resp, err)
})
```

Add `ListGrowthCampaignsReq` and `ListGrowthRewardGrantsReq` structs to `backend/app/internal/logic/admin/growth_campaign_logic.go` before adding these routes.

- [ ] **Step 4: Add route permission tests**

In `backend/app/internal/server/remaining_api_test.go`, add admin route cases:

```go
{name: "list growth campaigns", method: http.MethodGet, path: "/api/v1/admin/growth-campaigns"},
{name: "save growth campaign", method: http.MethodPost, path: "/api/v1/admin/growth-campaigns", body: `{"code":"starter_growth_2026_q3","name":"新手成长权益活动","status":"paused"}`},
```

- [ ] **Step 5: Run backend tests**

Run:

```bash
cd backend && go test ./...
```

Expected: PASS.

## Task 7: 小程序分享归因与轻提示

**Files:**
- Modify: `wxapp/common/resourceShare.js`
- Modify: `wxapp/common/resourceShare.test.mjs`
- Modify: `wxapp/pages/resource/detail.vue`
- Modify: `wxapp/pages/resource/detail.test.mjs`
- Modify: `wxapp/pages/vip/index.vue`
- Modify: `wxapp/pages/vip/index.test.mjs`

- [ ] **Step 1: Write failing share path test**

In `wxapp/common/resourceShare.test.mjs`, add:

```js
test('buildResourceSharePath includes share merchant source when present', () => {
  assert.equal(
    buildResourceSharePath({ id: 'resource-1', shareMerchantId: 'merchant-1' }),
    '/pages/resource/detail?id=resource-1&shareMerchantId=merchant-1'
  )
})
```

- [ ] **Step 2: Update share path**

In `wxapp/common/resourceShare.js`:

```js
export function buildResourceSharePath(resource = {}) {
  const id = normalizeText(resource.id)
  if (!id) return '/pages/home/index'
  const params = [`id=${encodeURIComponent(id)}`]
  const shareMerchantId = normalizeText(resource.shareMerchantId || resource.merchant?.id)
  if (shareMerchantId) {
    params.push(`shareMerchantId=${encodeURIComponent(shareMerchantId)}`)
  }
  return `/pages/resource/detail?${params.join('&')}`
}
```

- [ ] **Step 3: Write failing detail-page attribution test**

In `wxapp/pages/resource/detail.test.mjs`, add static checks:

```js
test('resource detail records share view only when share source exists', () => {
  const source = fs.readFileSync(path.join(root, 'pages/resource/detail.vue'), 'utf8')
  assert.match(source, /const shareMerchantId = ref\(''\)/)
  assert.match(source, /shareMerchantId\.value = options\.shareMerchantId \|\| ''/)
  assert.match(source, /recordShareView\(\)/)
  assert.match(source, /recordContact\('share_view'\)/)
})
```

- [ ] **Step 4: Add share_view on detail entry**

In `wxapp/pages/resource/detail.vue`, store `shareMerchantId` from route options and call `recordShareView()` after detail load. Guard it so normal direct entry does not record:

```js
const shareMerchantId = ref('')

function recordShareView() {
  if (!shareMerchantId.value || !resource.value.id) return
  recordContact('share_view').catch(() => {})
}
```

- [ ] **Step 5: Update entitlement page copy**

Replace merchant-card-centered copy with growth wording:

```text
发布优质资源、有效分享或开通 VIP，可获得更多发布和刷新权益
```

Do not add “完善商家名片得更多发布次数”.

- [ ] **Step 6: Run wxapp tests and build**

Run:

```bash
node --test wxapp/common/resourceShare.test.mjs wxapp/pages/resource/detail.test.mjs wxapp/pages/vip/index.test.mjs
npm --prefix wxapp run build:mp-weixin
```

Expected: tests PASS and build PASS.

## Task 8: 后台页面

**Files:**
- Create: `admin-web/src/api/growthCampaign.js`
- Create: `admin-web/src/views/GrowthCampaignView.vue`
- Modify: `admin-web/src/router/index.js`
- Modify: `admin-web/src/layouts/AdminLayout.vue`
- Create: `admin-web/scripts/growth-campaign-view.test.mjs`

- [ ] **Step 1: Write failing admin-web static test**

Create `admin-web/scripts/growth-campaign-view.test.mjs`:

```js
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'
import assert from 'node:assert/strict'

const root = path.resolve(import.meta.dirname, '..')

test('growth campaign admin page exposes pause disable and rule controls', () => {
  const view = fs.readFileSync(path.join(root, 'src/views/GrowthCampaignView.vue'), 'utf8')
  for (const token of ['增长活动', '暂停', '停用', '奖励数量', '有效期', '每日上限', '发放记录']) {
    assert.match(view, new RegExp(token))
  }
})

test('growth campaign menu and route are registered', () => {
  const router = fs.readFileSync(path.join(root, 'src/router/index.js'), 'utf8')
  const layout = fs.readFileSync(path.join(root, 'src/layouts/AdminLayout.vue'), 'utf8')
  assert.match(router, /growth-campaigns/)
  assert.match(layout, /增长活动/)
})
```

- [ ] **Step 2: Add API wrapper**

Create `admin-web/src/api/growthCampaign.js`:

```js
import http from './http'

export function listGrowthCampaigns(params = {}) {
  return http.get('/api/v1/admin/growth-campaigns', { params })
}

export function saveGrowthCampaign(data) {
  return http.post('/api/v1/admin/growth-campaigns', data)
}

export function saveGrowthRule(campaignCode, data) {
  return http.post(`/api/v1/admin/growth-campaigns/${campaignCode}/rules`, data)
}

export function listGrowthRewardGrants(campaignCode, params = {}) {
  return http.get(`/api/v1/admin/growth-campaigns/${campaignCode}/grants`, { params })
}
```

- [ ] **Step 3: Add view**

Create `admin-web/src/views/GrowthCampaignView.vue` with:

```vue
<template>
  <section>
    <div class="page-title">
      <h2>增长活动</h2>
      <el-button type="primary" @click="openCampaignDrawer">新增活动</el-button>
    </div>
    <section class="panel">
      <el-table :data="campaigns" stripe empty-text="暂无增长活动">
        <el-table-column prop="name" label="活动名称" min-width="180" />
        <el-table-column prop="code" label="编码" min-width="180" />
        <el-table-column prop="status" label="状态" width="100" />
        <el-table-column prop="startsAt" label="开始时间" width="180" />
        <el-table-column prop="endsAt" label="结束时间" width="180" />
        <el-table-column label="操作" width="260">
          <template #default="{ row }">
            <el-button link type="primary" @click="editCampaign(row)">编辑</el-button>
            <el-button link @click="pauseCampaign(row)">暂停</el-button>
            <el-button link type="danger" @click="disableCampaign(row)">停用</el-button>
          </template>
        </el-table-column>
      </el-table>
    </section>
    <section class="panel">
      <h3>规则配置</h3>
      <el-table :data="rules" stripe empty-text="请选择活动">
        <el-table-column prop="ruleCode" label="规则编码" min-width="180" />
        <el-table-column prop="triggerEvent" label="触发事件" min-width="180" />
        <el-table-column prop="rewardType" label="奖励类型" width="130" />
        <el-table-column prop="rewardAmount" label="奖励数量" width="110" />
        <el-table-column prop="validDays" label="有效期" width="100" />
        <el-table-column prop="perUserDailyLimit" label="每日上限" width="110" />
      </el-table>
    </section>
    <section class="panel">
      <h3>发放记录</h3>
      <el-table :data="grants" stripe empty-text="暂无发放记录">
        <el-table-column prop="merchantId" label="商家" min-width="160" />
        <el-table-column prop="ruleCode" label="规则" min-width="160" />
        <el-table-column prop="rewardType" label="权益" width="120" />
        <el-table-column prop="rewardAmount" label="数量" width="90" />
        <el-table-column prop="status" label="状态" width="100" />
        <el-table-column prop="createdAt" label="时间" width="180" />
      </el-table>
    </section>
  </section>
</template>
```

For edit drawers, copy the same `el-drawer` + `el-form` interaction shape used by `admin-web/src/views/VIPConfigView.vue`: one drawer for activity fields and one drawer for rule fields. Keep field labels exactly as “活动编码”“活动名称”“状态”“停用原因”“规则编码”“触发事件”“奖励类型”“奖励数量”“有效期”“每日上限”.

- [ ] **Step 4: Register route and menu**

Add route in `admin-web/src/router/index.js`:

```js
{
  path: '/growth-campaigns',
  name: 'growth-campaigns',
  component: () => import('../views/GrowthCampaignView.vue'),
}
```

Add menu item in `admin-web/src/layouts/AdminLayout.vue`:

```vue
<el-menu-item index="/growth-campaigns">增长活动</el-menu-item>
```

- [ ] **Step 5: Run admin-web checks**

Run:

```bash
node --test admin-web/scripts/growth-campaign-view.test.mjs
npm --prefix admin-web run build
```

Expected: tests PASS and build PASS.

## Task 9: 全量验证

**Files:**
- No new files.

- [ ] **Step 1: Run backend tests**

Run:

```bash
cd backend && go test ./...
```

Expected: PASS.

- [ ] **Step 2: Run backend migration static tests**

Run:

```bash
node --test backend/scripts/validate_migrations.test.mjs
```

Expected: PASS.

- [ ] **Step 3: Run wxapp tests**

Run:

```bash
find wxapp -name '*.test.mjs' -print | sort | xargs node --experimental-vm-modules --test
```

Expected: PASS.

- [ ] **Step 4: Build wxapp**

Run:

```bash
npm --prefix wxapp run build:mp-weixin
```

Expected: PASS. Existing Sass deprecation warnings are acceptable if no new compile errors appear.

- [ ] **Step 5: Run admin-web tests and build**

Run:

```bash
find admin-web/scripts -name '*.test.mjs' -print | sort | xargs node --test
npm --prefix admin-web run build
```

Expected: PASS.

## Implementation Notes

- 分享有效浏览奖励首期默认 `inactive`，先把 `share_view` 归因链路打通，运营确认数据质量后再启用。
- 邀请奖励首期默认 `inactive`，等邀请关系模型和来源归因上线后再启用。
- 增长权益发放失败不能阻断登录、审核、联系等主业务，只记录错误日志。
- 停用活动只停止新奖励发放，不回收已发放且未过期的权益。
- 任何后台停用操作都必须记录原因和操作人。
