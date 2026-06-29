# Apparel Industry Platform MVP Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the first testable MVP for the apparel industry resource platform, focused on the Zhili station vertical slice: publish resources, review them, browse/search them, view details, and contact verified suppliers.

**Architecture:** Use a small TypeScript monorepo with a Fastify API, Prisma data layer, Taro mobile mini-program client, and Vite React admin console. The first release keeps transactions, escrow, and in-app chat out of scope; WeChat/contact actions and manual matching are the operational bridge.

**Tech Stack:** npm workspaces, TypeScript, Fastify, Prisma, SQLite for local MVP, PostgreSQL-ready schema, Vitest, Taro React, Vite React admin, Zod.

---

## Scope

This plan implements the first engineering milestone from `docs/product/apparel-industry-platform-prd.md`.

Included:

- Zhili station as the first active city station.
- Four resource scenarios: find goods, clear inventory, find factories, accept orders.
- Auxiliary categories for recruitment, rental/transfer, logistics/material/service.
- Admin moderation workflow: draft, pending review, published, rejected, expired, removed.
- Business verification labels for factories, stalls, inventory, and service providers.
- Contact-click tracking for phone and WeChat copy actions.
- Seed data for local demo and user acceptance testing.

Excluded from this MVP:

- Escrow payment.
- Full e-commerce checkout.
- Real-time chat.
- Commission settlement.
- Multi-tenant city operators.
- Native app packages.

## File Structure

Create this repository structure:

```text
package.json
tsconfig.base.json
.gitignore
.env.example
apps/
  api/
    package.json
    tsconfig.json
    vitest.config.ts
    prisma/
      schema.prisma
      seed.ts
    src/
      server.ts
      app.ts
      config.ts
      db.ts
      modules/
        health/
          health.routes.ts
        catalogs/
          catalogs.data.ts
          catalogs.routes.ts
        resources/
          resources.schema.ts
          resources.repository.ts
          resources.routes.ts
          resources.test.ts
        verifications/
          verifications.schema.ts
          verifications.repository.ts
          verifications.routes.ts
          verifications.test.ts
        admin/
          admin.routes.ts
          admin.test.ts
  mobile/
    package.json
    project.config.json
    tsconfig.json
    config/
      index.ts
    src/
      app.config.ts
      app.tsx
      app.scss
      services/
        api.ts
      pages/
        home/
          index.config.ts
          index.tsx
          index.scss
        resources/
          index.config.ts
          index.tsx
          index.scss
        publish/
          index.config.ts
          index.tsx
          index.scss
        detail/
          index.config.ts
          index.tsx
          index.scss
  admin/
    package.json
    index.html
    tsconfig.json
    vite.config.ts
    src/
      main.tsx
      App.tsx
      api.ts
      styles.css
packages/
  shared/
    package.json
    tsconfig.json
    src/
      enums.ts
      resource.ts
      verification.ts
      index.ts
docs/
  product/
    apparel-industry-platform-prd.md
  superpowers/
    plans/
      2026-06-26-apparel-platform-mvp.md
```

Boundary decisions:

- `packages/shared` owns enums and DTO types shared by API, mobile, and admin.
- `apps/api` owns persistence, validation, routes, and moderation.
- `apps/mobile` owns user-facing browsing, publishing, and contact actions.
- `apps/admin` owns internal review and verification operations.

## Task 1: Scaffold The Monorepo

**Files:**
- Create: `package.json`
- Create: `tsconfig.base.json`
- Create: `.gitignore`
- Create: `.env.example`
- Create: `packages/shared/package.json`
- Create: `packages/shared/tsconfig.json`
- Create: `packages/shared/src/enums.ts`
- Create: `packages/shared/src/resource.ts`
- Create: `packages/shared/src/verification.ts`
- Create: `packages/shared/src/index.ts`

- [ ] **Step 1: Create root workspace files**

Write `package.json`:

```json
{
  "name": "wplink",
  "private": true,
  "version": "0.1.0",
  "workspaces": [
    "apps/*",
    "packages/*"
  ],
  "scripts": {
    "build": "npm run build --workspaces --if-present",
    "test": "npm run test --workspaces --if-present",
    "typecheck": "npm run typecheck --workspaces --if-present",
    "dev:api": "npm --workspace apps/api run dev",
    "dev:admin": "npm --workspace apps/admin run dev",
    "dev:mobile": "npm --workspace apps/mobile run dev:weapp"
  },
  "devDependencies": {
    "typescript": "^5.5.4"
  }
}
```

Write `tsconfig.base.json`:

```json
{
  "compilerOptions": {
    "target": "ES2022",
    "module": "ESNext",
    "moduleResolution": "Bundler",
    "strict": true,
    "esModuleInterop": true,
    "forceConsistentCasingInFileNames": true,
    "skipLibCheck": true,
    "resolveJsonModule": true,
    "baseUrl": ".",
    "paths": {
      "@wplink/shared": ["packages/shared/src/index.ts"]
    }
  }
}
```

Write `.gitignore`:

```gitignore
node_modules
dist
.turbo
.DS_Store
.env
apps/api/prisma/dev.db
apps/api/prisma/dev.db-journal
```

Write `.env.example`:

```bash
DATABASE_URL="file:./dev.db"
API_PORT=4000
VITE_API_BASE_URL="http://127.0.0.1:4000"
TARO_APP_API_BASE_URL="http://127.0.0.1:4000"
```

- [ ] **Step 2: Create shared package**

Write `packages/shared/package.json`:

```json
{
  "name": "@wplink/shared",
  "version": "0.1.0",
  "type": "module",
  "main": "dist/index.js",
  "types": "dist/index.d.ts",
  "scripts": {
    "build": "tsc -p tsconfig.json",
    "typecheck": "tsc -p tsconfig.json --noEmit"
  }
}
```

Write `packages/shared/tsconfig.json`:

```json
{
  "extends": "../../tsconfig.base.json",
  "compilerOptions": {
    "outDir": "dist",
    "declaration": true,
    "rootDir": "src"
  },
  "include": ["src"]
}
```

Write `packages/shared/src/enums.ts`:

```ts
export const cityCodes = ["zhili", "guangzhou", "humen", "hangzhou", "changshu"] as const;
export type CityCode = (typeof cityCodes)[number];

export const resourceTypes = [
  "goods",
  "inventory",
  "factory",
  "order",
  "job",
  "rental",
  "service"
] as const;
export type ResourceType = (typeof resourceTypes)[number];

export const resourceStatuses = ["draft", "pending", "published", "rejected", "expired", "removed"] as const;
export type ResourceStatus = (typeof resourceStatuses)[number];

export const verificationTypes = ["factory", "stall", "inventory", "serviceProvider"] as const;
export type VerificationType = (typeof verificationTypes)[number];

export const verificationStatuses = ["pending", "approved", "rejected", "revoked"] as const;
export type VerificationStatus = (typeof verificationStatuses)[number];
```

Write `packages/shared/src/resource.ts`:

```ts
import type { CityCode, ResourceStatus, ResourceType } from "./enums";

export type ContactInfo = {
  name: string;
  phone: string;
  wechat?: string;
};

export type ResourceListItem = {
  id: string;
  type: ResourceType;
  status: ResourceStatus;
  city: CityCode;
  title: string;
  category: string;
  district?: string;
  priceText?: string;
  quantityText?: string;
  coverUrl?: string;
  isVerified: boolean;
  publishedAt?: string;
  refreshedAt?: string;
};

export type ResourceDetail = ResourceListItem & {
  description: string;
  tags: string[];
  images: string[];
  contact: ContactInfo;
  attributes: Record<string, string | number | boolean>;
};
```

Write `packages/shared/src/verification.ts`:

```ts
import type { VerificationStatus, VerificationType } from "./enums";

export type VerificationSummary = {
  id: string;
  ownerName: string;
  type: VerificationType;
  status: VerificationStatus;
  city: string;
  submittedAt: string;
  reviewedAt?: string;
};
```

Write `packages/shared/src/index.ts`:

```ts
export * from "./enums";
export * from "./resource";
export * from "./verification";
```

- [ ] **Step 3: Install dependencies**

Run:

```bash
npm install
```

Expected: npm installs TypeScript and creates `package-lock.json`.

- [ ] **Step 4: Run shared package typecheck**

Run:

```bash
npm --workspace packages/shared run typecheck
```

Expected: exit code `0`.

- [ ] **Step 5: Commit**

```bash
git add package.json package-lock.json tsconfig.base.json .gitignore .env.example packages/shared
git commit -m "chore: scaffold apparel platform workspace"
```

## Task 2: Build API Data Model

**Files:**
- Create: `apps/api/package.json`
- Create: `apps/api/tsconfig.json`
- Create: `apps/api/vitest.config.ts`
- Create: `apps/api/prisma/schema.prisma`
- Create: `apps/api/prisma/seed.ts`
- Create: `apps/api/src/config.ts`
- Create: `apps/api/src/db.ts`

- [ ] **Step 1: Create API package files**

Write `apps/api/package.json`:

```json
{
  "name": "@wplink/api",
  "version": "0.1.0",
  "type": "module",
  "scripts": {
    "dev": "tsx watch src/server.ts",
    "build": "tsc -p tsconfig.json",
    "start": "node dist/server.js",
    "test": "vitest run",
    "typecheck": "tsc -p tsconfig.json --noEmit",
    "prisma:generate": "prisma generate",
    "prisma:migrate": "prisma migrate dev",
    "prisma:seed": "tsx prisma/seed.ts"
  },
  "dependencies": {
    "@fastify/cors": "^9.0.1",
    "@prisma/client": "^5.18.0",
    "@wplink/shared": "file:../../packages/shared",
    "fastify": "^4.28.1",
    "zod": "^3.23.8"
  },
  "devDependencies": {
    "@types/node": "^22.5.0",
    "prisma": "^5.18.0",
    "tsx": "^4.19.0",
    "vitest": "^2.0.5"
  },
  "prisma": {
    "seed": "tsx prisma/seed.ts"
  }
}
```

Write `apps/api/tsconfig.json`:

```json
{
  "extends": "../../tsconfig.base.json",
  "compilerOptions": {
    "outDir": "dist",
    "rootDir": ".",
    "types": ["node", "vitest/globals"]
  },
  "include": ["src", "prisma"]
}
```

Write `apps/api/vitest.config.ts`:

```ts
import { defineConfig } from "vitest/config";

export default defineConfig({
  test: {
    environment: "node",
    globals: true
  }
});
```

- [ ] **Step 2: Create Prisma schema**

Write `apps/api/prisma/schema.prisma`:

```prisma
generator client {
  provider = "prisma-client-js"
}

datasource db {
  provider = "sqlite"
  url      = env("DATABASE_URL")
}

model Resource {
  id           String   @id @default(cuid())
  type         String
  status       String   @default("pending")
  city         String
  title        String
  category     String
  district     String?
  priceText    String?
  quantityText String?
  coverUrl     String?
  description  String
  tagsJson     String   @default("[]")
  imagesJson   String   @default("[]")
  attrsJson    String   @default("{}")
  contactName  String
  contactPhone String
  contactWechat String?
  isVerified   Boolean  @default(false)
  rejectReason String?
  publishedAt  DateTime?
  refreshedAt  DateTime?
  expiresAt    DateTime?
  createdAt    DateTime @default(now())
  updatedAt    DateTime @updatedAt

  contactEvents ContactEvent[]
}

model Verification {
  id             String   @id @default(cuid())
  ownerName      String
  type           String
  status         String   @default("pending")
  city           String
  phone          String
  businessName   String?
  licenseUrl     String?
  storefrontUrl  String?
  locationText   String?
  reviewNote     String?
  submittedAt    DateTime @default(now())
  reviewedAt     DateTime?
  createdAt      DateTime @default(now())
  updatedAt      DateTime @updatedAt
}

model ContactEvent {
  id          String   @id @default(cuid())
  resourceId  String
  action      String
  createdAt   DateTime @default(now())

  resource Resource @relation(fields: [resourceId], references: [id])
}
```

- [ ] **Step 3: Create config and database helpers**

Write `apps/api/src/config.ts`:

```ts
export type AppConfig = {
  port: number;
};

export function loadConfig(env = process.env): AppConfig {
  return {
    port: Number(env.API_PORT ?? 4000)
  };
}
```

Write `apps/api/src/db.ts`:

```ts
import { PrismaClient } from "@prisma/client";

export const prisma = new PrismaClient();
```

- [ ] **Step 4: Create seed data**

Write `apps/api/prisma/seed.ts`:

```ts
import { PrismaClient } from "@prisma/client";

const prisma = new PrismaClient();

const now = new Date();
const inSevenDays = new Date(now.getTime() + 7 * 24 * 60 * 60 * 1000);

await prisma.contactEvent.deleteMany();
await prisma.resource.deleteMany();
await prisma.verification.deleteMany();

await prisma.resource.createMany({
  data: [
    {
      type: "inventory",
      status: "published",
      city: "zhili",
      title: "织里女童春款卫衣库存整包清",
      category: "童装",
      district: "织里童装城周边",
      priceText: "打包 18 元/件",
      quantityText: "3200 件",
      description: "女童春款卫衣，90-140 码，整包优先，可现场看货。",
      tagsJson: JSON.stringify(["急清", "支持看货", "可直播"]),
      imagesJson: JSON.stringify([]),
      attrsJson: JSON.stringify({
        season: "春款",
        sizeRange: "90-140",
        allowSample: true,
        allowLiveSale: true
      }),
      contactName: "张老板",
      contactPhone: "13800000001",
      contactWechat: "zhili-stock-001",
      isVerified: true,
      publishedAt: now,
      refreshedAt: now,
      expiresAt: inSevenDays
    },
    {
      type: "factory",
      status: "published",
      city: "zhili",
      title: "童装卫衣工厂有空档期，可接小单快反",
      category: "加工厂",
      district: "织里镇",
      priceText: "按款报价",
      quantityText: "日产 1200 件",
      description: "主做中小童卫衣、套装，有裁剪和后道资源，支持打样。",
      tagsJson: JSON.stringify(["小单快反", "可打样", "有空档"]),
      imagesJson: JSON.stringify([]),
      attrsJson: JSON.stringify({
        capacityPerDay: 1200,
        minOrderQuantity: 300,
        canProvideMaterials: false
      }),
      contactName: "李厂长",
      contactPhone: "13800000002",
      contactWechat: "zhili-factory-002",
      isVerified: true,
      publishedAt: now,
      refreshedAt: now,
      expiresAt: inSevenDays
    }
  ]
});

await prisma.verification.create({
  data: {
    ownerName: "李厂长",
    type: "factory",
    status: "approved",
    city: "zhili",
    phone: "13800000002",
    businessName: "织里样板童装加工厂",
    locationText: "湖州织里镇"
  }
});

await prisma.$disconnect();
```

- [ ] **Step 5: Install API dependencies and initialize database**

Run:

```bash
npm install
npm --workspace apps/api run prisma:generate
npm --workspace apps/api run prisma:migrate -- --name init
npm --workspace apps/api run prisma:seed
```

Expected:

- Prisma client generated.
- SQLite database created at `apps/api/prisma/dev.db`.
- Seed script exits with code `0`.

- [ ] **Step 6: Commit**

```bash
git add apps/api package.json package-lock.json
git commit -m "feat: add apparel platform data model"
```

## Task 3: Implement Resource API

**Files:**
- Create: `apps/api/src/modules/resources/resources.schema.ts`
- Create: `apps/api/src/modules/resources/resources.repository.ts`
- Create: `apps/api/src/modules/resources/resources.routes.ts`
- Create: `apps/api/src/modules/resources/resources.test.ts`
- Create: `apps/api/src/app.ts`
- Create: `apps/api/src/server.ts`
- Create: `apps/api/src/modules/health/health.routes.ts`
- Create: `apps/api/src/modules/catalogs/catalogs.data.ts`
- Create: `apps/api/src/modules/catalogs/catalogs.routes.ts`

- [ ] **Step 1: Write failing resource API tests**

Write `apps/api/src/modules/resources/resources.test.ts`:

```ts
import { buildApp } from "../../app";

describe("resource routes", () => {
  it("creates a pending inventory resource", async () => {
    const app = await buildApp();

    const response = await app.inject({
      method: "POST",
      url: "/resources",
      payload: {
        type: "inventory",
        city: "zhili",
        title: "男童夏款短裤库存",
        category: "童装",
        description: "男童夏款短裤，100-150 码，整包优先。",
        contact: {
          name: "王老板",
          phone: "13800000003",
          wechat: "stock003"
        },
        attributes: {
          season: "夏款",
          sizeRange: "100-150"
        }
      }
    });

    expect(response.statusCode).toBe(201);
    const body = response.json();
    expect(body.status).toBe("pending");
    expect(body.title).toBe("男童夏款短裤库存");
  });

  it("lists only published resources by default", async () => {
    const app = await buildApp();

    const response = await app.inject({
      method: "GET",
      url: "/resources?city=zhili"
    });

    expect(response.statusCode).toBe(200);
    const body = response.json();
    expect(Array.isArray(body.items)).toBe(true);
    expect(body.items.every((item: { status: string }) => item.status === "published")).toBe(true);
  });
});
```

- [ ] **Step 2: Run tests to verify failure**

Run:

```bash
npm --workspace apps/api run test -- resources.test.ts
```

Expected: FAIL because `buildApp` and resource routes do not exist.

- [ ] **Step 3: Add route validation and repository**

Write `apps/api/src/modules/resources/resources.schema.ts`:

```ts
import { z } from "zod";
import { cityCodes, resourceTypes } from "@wplink/shared";

export const contactSchema = z.object({
  name: z.string().min(1),
  phone: z.string().min(6),
  wechat: z.string().optional()
});

export const createResourceSchema = z.object({
  type: z.enum(resourceTypes),
  city: z.enum(cityCodes),
  title: z.string().min(2).max(80),
  category: z.string().min(1).max(30),
  district: z.string().max(40).optional(),
  priceText: z.string().max(40).optional(),
  quantityText: z.string().max(40).optional(),
  coverUrl: z.string().url().optional(),
  description: z.string().min(5).max(1000),
  tags: z.array(z.string().max(20)).max(10).default([]),
  images: z.array(z.string().url()).max(9).default([]),
  contact: contactSchema,
  attributes: z.record(z.union([z.string(), z.number(), z.boolean()])).default({})
});

export const listResourceQuerySchema = z.object({
  city: z.enum(cityCodes).optional(),
  type: z.enum(resourceTypes).optional(),
  category: z.string().optional(),
  keyword: z.string().optional(),
  includePending: z.coerce.boolean().default(false)
});
```

Write `apps/api/src/modules/resources/resources.repository.ts`:

```ts
import type { PrismaClient, Resource } from "@prisma/client";
import type { ResourceDetail, ResourceListItem } from "@wplink/shared";
import type { z } from "zod";
import type { createResourceSchema, listResourceQuerySchema } from "./resources.schema";

type CreateResourceInput = z.infer<typeof createResourceSchema>;
type ListResourceQuery = z.infer<typeof listResourceQuerySchema>;

function parseJson<T>(value: string, fallback: T): T {
  try {
    return JSON.parse(value) as T;
  } catch {
    return fallback;
  }
}

function toListItem(resource: Resource): ResourceListItem {
  return {
    id: resource.id,
    type: resource.type as ResourceListItem["type"],
    status: resource.status as ResourceListItem["status"],
    city: resource.city as ResourceListItem["city"],
    title: resource.title,
    category: resource.category,
    district: resource.district ?? undefined,
    priceText: resource.priceText ?? undefined,
    quantityText: resource.quantityText ?? undefined,
    coverUrl: resource.coverUrl ?? undefined,
    isVerified: resource.isVerified,
    publishedAt: resource.publishedAt?.toISOString(),
    refreshedAt: resource.refreshedAt?.toISOString()
  };
}

function toDetail(resource: Resource): ResourceDetail {
  return {
    ...toListItem(resource),
    description: resource.description,
    tags: parseJson<string[]>(resource.tagsJson, []),
    images: parseJson<string[]>(resource.imagesJson, []),
    contact: {
      name: resource.contactName,
      phone: resource.contactPhone,
      wechat: resource.contactWechat ?? undefined
    },
    attributes: parseJson<Record<string, string | number | boolean>>(resource.attrsJson, {})
  };
}

export function createResourceRepository(prisma: PrismaClient) {
  return {
    async create(input: CreateResourceInput): Promise<ResourceDetail> {
      const resource = await prisma.resource.create({
        data: {
          type: input.type,
          city: input.city,
          title: input.title,
          category: input.category,
          district: input.district,
          priceText: input.priceText,
          quantityText: input.quantityText,
          coverUrl: input.coverUrl,
          description: input.description,
          tagsJson: JSON.stringify(input.tags),
          imagesJson: JSON.stringify(input.images),
          attrsJson: JSON.stringify(input.attributes),
          contactName: input.contact.name,
          contactPhone: input.contact.phone,
          contactWechat: input.contact.wechat,
          status: "pending"
        }
      });

      return toDetail(resource);
    },

    async list(query: ListResourceQuery): Promise<ResourceListItem[]> {
      const resources = await prisma.resource.findMany({
        where: {
          city: query.city,
          type: query.type,
          category: query.category,
          status: query.includePending ? undefined : "published",
          OR: query.keyword
            ? [
                { title: { contains: query.keyword } },
                { description: { contains: query.keyword } },
                { category: { contains: query.keyword } }
              ]
            : undefined
        },
        orderBy: [{ isVerified: "desc" }, { refreshedAt: "desc" }, { createdAt: "desc" }],
        take: 100
      });

      return resources.map(toListItem);
    },

    async getById(id: string): Promise<ResourceDetail | null> {
      const resource = await prisma.resource.findUnique({ where: { id } });
      return resource ? toDetail(resource) : null;
    },

    async trackContact(resourceId: string, action: "phone" | "wechat"): Promise<void> {
      await prisma.contactEvent.create({
        data: { resourceId, action }
      });
    }
  };
}
```

- [ ] **Step 4: Add routes and app builder**

Write `apps/api/src/modules/resources/resources.routes.ts`:

```ts
import type { FastifyInstance } from "fastify";
import { prisma } from "../../db";
import { createResourceRepository } from "./resources.repository";
import { createResourceSchema, listResourceQuerySchema } from "./resources.schema";

export async function registerResourceRoutes(app: FastifyInstance) {
  const repository = createResourceRepository(prisma);

  app.post("/resources", async (request, reply) => {
    const input = createResourceSchema.parse(request.body);
    const resource = await repository.create(input);
    return reply.code(201).send(resource);
  });

  app.get("/resources", async (request) => {
    const query = listResourceQuerySchema.parse(request.query);
    const items = await repository.list(query);
    return { items };
  });

  app.get("/resources/:id", async (request, reply) => {
    const { id } = request.params as { id: string };
    const resource = await repository.getById(id);

    if (!resource) {
      return reply.code(404).send({ message: "资源不存在或已下架" });
    }

    return resource;
  });

  app.post("/resources/:id/contact-events", async (request, reply) => {
    const { id } = request.params as { id: string };
    const body = request.body as { action?: "phone" | "wechat" };

    if (body.action !== "phone" && body.action !== "wechat") {
      return reply.code(400).send({ message: "联系方式动作无效" });
    }

    await repository.trackContact(id, body.action);
    return reply.code(204).send();
  });
}
```

Write `apps/api/src/modules/health/health.routes.ts`:

```ts
import type { FastifyInstance } from "fastify";

export async function registerHealthRoutes(app: FastifyInstance) {
  app.get("/health", async () => ({ ok: true }));
}
```

Write `apps/api/src/modules/catalogs/catalogs.data.ts`:

```ts
export const catalogs = {
  cities: [
    { code: "zhili", name: "织里", primaryCategory: "童装" },
    { code: "guangzhou", name: "广州", primaryCategory: "女装" },
    { code: "humen", name: "虎门", primaryCategory: "女装/加工" },
    { code: "hangzhou", name: "杭州", primaryCategory: "电商女装" },
    { code: "changshu", name: "常熟", primaryCategory: "男装" }
  ],
  resourceTypes: [
    { code: "goods", name: "找货" },
    { code: "inventory", name: "清库存" },
    { code: "factory", name: "找厂" },
    { code: "order", name: "接订单" },
    { code: "job", name: "招聘" },
    { code: "rental", name: "出租/转让" },
    { code: "service", name: "配套服务" }
  ],
  zhiliCategories: ["童装", "加工厂", "库存尾货", "后道", "印花绣花", "物流", "招聘", "厂房出租"]
};
```

Write `apps/api/src/modules/catalogs/catalogs.routes.ts`:

```ts
import type { FastifyInstance } from "fastify";
import { catalogs } from "./catalogs.data";

export async function registerCatalogRoutes(app: FastifyInstance) {
  app.get("/catalogs", async () => catalogs);
}
```

Write `apps/api/src/app.ts`:

```ts
import cors from "@fastify/cors";
import Fastify from "fastify";
import { ZodError } from "zod";
import { registerCatalogRoutes } from "./modules/catalogs/catalogs.routes";
import { registerHealthRoutes } from "./modules/health/health.routes";
import { registerResourceRoutes } from "./modules/resources/resources.routes";

export async function buildApp() {
  const app = Fastify({ logger: false });

  await app.register(cors, { origin: true });

  app.setErrorHandler((error, _request, reply) => {
    if (error instanceof ZodError) {
      return reply.code(400).send({
        message: "提交内容不完整或格式不正确",
        issues: error.issues
      });
    }

    return reply.code(500).send({ message: "服务暂时不可用，请稍后再试" });
  });

  await registerHealthRoutes(app);
  await registerCatalogRoutes(app);
  await registerResourceRoutes(app);

  return app;
}
```

Write `apps/api/src/server.ts`:

```ts
import { buildApp } from "./app";
import { loadConfig } from "./config";

const config = loadConfig();
const app = await buildApp();

await app.listen({ port: config.port, host: "0.0.0.0" });
```

- [ ] **Step 5: Run resource tests**

Run:

```bash
npm --workspace apps/api run test -- resources.test.ts
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add apps/api/src
git commit -m "feat: add resource publishing api"
```

## Task 4: Implement Admin Moderation And Verification API

**Files:**
- Create: `apps/api/src/modules/admin/admin.routes.ts`
- Create: `apps/api/src/modules/admin/admin.test.ts`
- Create: `apps/api/src/modules/verifications/verifications.schema.ts`
- Create: `apps/api/src/modules/verifications/verifications.repository.ts`
- Create: `apps/api/src/modules/verifications/verifications.routes.ts`
- Create: `apps/api/src/modules/verifications/verifications.test.ts`
- Modify: `apps/api/src/app.ts`

- [ ] **Step 1: Write admin tests**

Write `apps/api/src/modules/admin/admin.test.ts`:

```ts
import { buildApp } from "../../app";

describe("admin moderation routes", () => {
  it("publishes a pending resource", async () => {
    const app = await buildApp();

    const created = await app.inject({
      method: "POST",
      url: "/resources",
      payload: {
        type: "goods",
        city: "zhili",
        title: "中小童卫衣现货",
        category: "童装",
        description: "中小童卫衣现货，支持看样和长期合作。",
        contact: { name: "陈老板", phone: "13800000004" }
      }
    });

    const id = created.json().id;
    const reviewed = await app.inject({
      method: "POST",
      url: `/admin/resources/${id}/review`,
      payload: { action: "publish" }
    });

    expect(reviewed.statusCode).toBe(200);
    expect(reviewed.json().status).toBe("published");
    expect(reviewed.json().publishedAt).toBeTruthy();
  });
});
```

- [ ] **Step 2: Write verification tests**

Write `apps/api/src/modules/verifications/verifications.test.ts`:

```ts
import { buildApp } from "../../app";

describe("verification routes", () => {
  it("creates a pending factory verification", async () => {
    const app = await buildApp();

    const response = await app.inject({
      method: "POST",
      url: "/verifications",
      payload: {
        ownerName: "李厂长",
        type: "factory",
        city: "zhili",
        phone: "13800000005",
        businessName: "织里快反童装厂",
        locationText: "织里镇"
      }
    });

    expect(response.statusCode).toBe(201);
    expect(response.json().status).toBe("pending");
  });
});
```

- [ ] **Step 3: Run tests to verify failure**

Run:

```bash
npm --workspace apps/api run test -- admin.test.ts verifications.test.ts
```

Expected: FAIL because admin and verification routes are not registered.

- [ ] **Step 4: Add admin moderation route**

Write `apps/api/src/modules/admin/admin.routes.ts`:

```ts
import type { FastifyInstance } from "fastify";
import { z } from "zod";
import { prisma } from "../../db";

const reviewSchema = z.object({
  action: z.enum(["publish", "reject", "remove"]),
  reason: z.string().max(200).optional()
});

export async function registerAdminRoutes(app: FastifyInstance) {
  app.get("/admin/resources", async (request) => {
    const query = request.query as { status?: string };
    const resources = await prisma.resource.findMany({
      where: { status: query.status },
      orderBy: { createdAt: "desc" },
      take: 100
    });

    return { items: resources };
  });

  app.post("/admin/resources/:id/review", async (request, reply) => {
    const { id } = request.params as { id: string };
    const input = reviewSchema.parse(request.body);
    const now = new Date();
    const expiresAt = new Date(now.getTime() + 7 * 24 * 60 * 60 * 1000);

    const status = input.action === "publish" ? "published" : input.action === "reject" ? "rejected" : "removed";

    const resource = await prisma.resource.update({
      where: { id },
      data: {
        status,
        rejectReason: input.action === "reject" ? input.reason ?? "信息未通过审核" : null,
        publishedAt: input.action === "publish" ? now : undefined,
        refreshedAt: input.action === "publish" ? now : undefined,
        expiresAt: input.action === "publish" ? expiresAt : undefined
      }
    });

    return reply.send(resource);
  });
}
```

- [ ] **Step 5: Add verification route**

Write `apps/api/src/modules/verifications/verifications.schema.ts`:

```ts
import { z } from "zod";
import { cityCodes, verificationTypes } from "@wplink/shared";

export const createVerificationSchema = z.object({
  ownerName: z.string().min(1).max(40),
  type: z.enum(verificationTypes),
  city: z.enum(cityCodes),
  phone: z.string().min(6).max(30),
  businessName: z.string().max(80).optional(),
  licenseUrl: z.string().url().optional(),
  storefrontUrl: z.string().url().optional(),
  locationText: z.string().max(120).optional()
});
```

Write `apps/api/src/modules/verifications/verifications.repository.ts`:

```ts
import type { PrismaClient } from "@prisma/client";
import type { VerificationSummary } from "@wplink/shared";
import type { z } from "zod";
import type { createVerificationSchema } from "./verifications.schema";

type CreateVerificationInput = z.infer<typeof createVerificationSchema>;

function toSummary(item: {
  id: string;
  ownerName: string;
  type: string;
  status: string;
  city: string;
  submittedAt: Date;
  reviewedAt: Date | null;
}): VerificationSummary {
  return {
    id: item.id,
    ownerName: item.ownerName,
    type: item.type as VerificationSummary["type"],
    status: item.status as VerificationSummary["status"],
    city: item.city,
    submittedAt: item.submittedAt.toISOString(),
    reviewedAt: item.reviewedAt?.toISOString()
  };
}

export function createVerificationRepository(prisma: PrismaClient) {
  return {
    async create(input: CreateVerificationInput): Promise<VerificationSummary> {
      const verification = await prisma.verification.create({
        data: input
      });

      return toSummary(verification);
    },

    async listPending(): Promise<VerificationSummary[]> {
      const items = await prisma.verification.findMany({
        where: { status: "pending" },
        orderBy: { submittedAt: "desc" }
      });

      return items.map(toSummary);
    }
  };
}
```

Write `apps/api/src/modules/verifications/verifications.routes.ts`:

```ts
import type { FastifyInstance } from "fastify";
import { prisma } from "../../db";
import { createVerificationRepository } from "./verifications.repository";
import { createVerificationSchema } from "./verifications.schema";

export async function registerVerificationRoutes(app: FastifyInstance) {
  const repository = createVerificationRepository(prisma);

  app.post("/verifications", async (request, reply) => {
    const input = createVerificationSchema.parse(request.body);
    const verification = await repository.create(input);
    return reply.code(201).send(verification);
  });

  app.get("/admin/verifications/pending", async () => {
    const items = await repository.listPending();
    return { items };
  });
}
```

- [ ] **Step 6: Register routes in app**

Modify `apps/api/src/app.ts` to include these imports:

```ts
import { registerAdminRoutes } from "./modules/admin/admin.routes";
import { registerVerificationRoutes } from "./modules/verifications/verifications.routes";
```

Update `buildApp()` route registration block:

```ts
  await registerHealthRoutes(app);
  await registerCatalogRoutes(app);
  await registerResourceRoutes(app);
  await registerVerificationRoutes(app);
  await registerAdminRoutes(app);
```

- [ ] **Step 7: Run admin and verification tests**

Run:

```bash
npm --workspace apps/api run test -- admin.test.ts verifications.test.ts
```

Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add apps/api/src/modules/admin apps/api/src/modules/verifications apps/api/src/app.ts
git commit -m "feat: add moderation and verification api"
```

## Task 5: Build Mobile Mini-Program MVP

**Files:**
- Create: `apps/mobile/package.json`
- Create: `apps/mobile/project.config.json`
- Create: `apps/mobile/tsconfig.json`
- Create: `apps/mobile/config/index.ts`
- Create: `apps/mobile/src/app.config.ts`
- Create: `apps/mobile/src/app.tsx`
- Create: `apps/mobile/src/app.scss`
- Create: `apps/mobile/src/services/api.ts`
- Create: `apps/mobile/src/pages/home/index.config.ts`
- Create: `apps/mobile/src/pages/home/index.tsx`
- Create: `apps/mobile/src/pages/home/index.scss`
- Create: `apps/mobile/src/pages/resources/index.config.ts`
- Create: `apps/mobile/src/pages/resources/index.tsx`
- Create: `apps/mobile/src/pages/resources/index.scss`
- Create: `apps/mobile/src/pages/detail/index.config.ts`
- Create: `apps/mobile/src/pages/detail/index.tsx`
- Create: `apps/mobile/src/pages/detail/index.scss`
- Create: `apps/mobile/src/pages/publish/index.config.ts`
- Create: `apps/mobile/src/pages/publish/index.tsx`
- Create: `apps/mobile/src/pages/publish/index.scss`

- [ ] **Step 1: Create mobile package**

Write `apps/mobile/package.json`:

```json
{
  "name": "@wplink/mobile",
  "version": "0.1.0",
  "private": true,
  "scripts": {
    "dev:weapp": "taro build --type weapp --watch",
    "build:weapp": "taro build --type weapp",
    "typecheck": "tsc -p tsconfig.json --noEmit"
  },
  "dependencies": {
    "@tarojs/components": "^3.6.35",
    "@tarojs/helper": "^3.6.35",
    "@tarojs/plugin-framework-react": "^3.6.35",
    "@tarojs/plugin-platform-weapp": "^3.6.35",
    "@tarojs/react": "^3.6.35",
    "@tarojs/runtime": "^3.6.35",
    "@tarojs/taro": "^3.6.35",
    "@wplink/shared": "file:../../packages/shared",
    "react": "^18.3.1",
    "react-dom": "^18.3.1"
  },
  "devDependencies": {
    "@tarojs/cli": "^3.6.35",
    "@types/react": "^18.3.4"
  }
}
```

Write `apps/mobile/tsconfig.json`:

```json
{
  "extends": "../../tsconfig.base.json",
  "compilerOptions": {
    "jsx": "react-jsx",
    "types": ["@tarojs/taro"]
  },
  "include": ["config", "src"]
}
```

Write `apps/mobile/project.config.json`:

```json
{
  "miniprogramRoot": "dist/",
  "projectname": "wplink-mobile",
  "description": "服装产业带资源撮合平台小程序 MVP",
  "appid": "touristappid",
  "setting": {
    "urlCheck": false,
    "es6": true,
    "enhance": true,
    "postcss": true,
    "minified": false
  },
  "compileType": "miniprogram"
}
```

Write `apps/mobile/config/index.ts`:

```ts
import { defineConfig } from "@tarojs/cli";

export default defineConfig({
  projectName: "wplink-mobile",
  date: "2026-06-26",
  designWidth: 750,
  deviceRatio: {
    640: 2.34,
    750: 1,
    828: 1.81
  },
  sourceRoot: "src",
  outputRoot: "dist",
  framework: "react",
  compiler: "webpack5",
  mini: {},
  h5: {}
});
```

- [ ] **Step 2: Add app shell and API service**

Write `apps/mobile/src/app.config.ts`:

```ts
export default defineAppConfig({
  pages: [
    "pages/home/index",
    "pages/resources/index",
    "pages/detail/index",
    "pages/publish/index"
  ],
  window: {
    backgroundTextStyle: "light",
    navigationBarBackgroundColor: "#111827",
    navigationBarTitleText: "衣货通·织里站",
    navigationBarTextStyle: "white"
  }
});
```

Write `apps/mobile/src/app.tsx`:

```tsx
import "./app.scss";

export default function App(props: { children: React.ReactNode }) {
  return props.children;
}
```

Write `apps/mobile/src/app.scss`:

```scss
page {
  background: #f6f7f9;
  color: #111827;
  font-family: -apple-system, BlinkMacSystemFont, "PingFang SC", sans-serif;
}

.page {
  min-height: 100vh;
  padding: 24px;
  box-sizing: border-box;
}
```

Write `apps/mobile/src/services/api.ts`:

```ts
import Taro from "@tarojs/taro";
import type { ResourceDetail, ResourceListItem, ResourceType } from "@wplink/shared";

const baseUrl = process.env.TARO_APP_API_BASE_URL ?? "http://127.0.0.1:4000";

export async function listResources(params: { type?: ResourceType; city?: string } = {}) {
  const query = Object.entries(params)
    .filter(([, value]) => value)
    .map(([key, value]) => `${key}=${encodeURIComponent(String(value))}`)
    .join("&");

  const response = await Taro.request<{ items: ResourceListItem[] }>({
    url: `${baseUrl}/resources${query ? `?${query}` : ""}`,
    method: "GET"
  });

  return response.data.items;
}

export async function getResource(id: string) {
  const response = await Taro.request<ResourceDetail>({
    url: `${baseUrl}/resources/${id}`,
    method: "GET"
  });

  return response.data;
}

export async function createResource(payload: unknown) {
  const response = await Taro.request<ResourceDetail>({
    url: `${baseUrl}/resources`,
    method: "POST",
    data: payload
  });

  return response.data;
}

export async function trackContact(id: string, action: "phone" | "wechat") {
  await Taro.request({
    url: `${baseUrl}/resources/${id}/contact-events`,
    method: "POST",
    data: { action }
  });
}
```

- [ ] **Step 3: Build home page**

Write `apps/mobile/src/pages/home/index.config.ts`:

```ts
export default definePageConfig({
  navigationBarTitleText: "衣货通·织里站"
});
```

Write `apps/mobile/src/pages/home/index.tsx`:

```tsx
import Taro from "@tarojs/taro";
import { Text, View } from "@tarojs/components";
import "./index.scss";

const actions = [
  { title: "我要找货", type: "goods", desc: "源头货、现货、直播货盘" },
  { title: "我要清货", type: "inventory", desc: "库存、尾货、换季货" },
  { title: "我要找厂", type: "factory", desc: "加工厂、空档产能、配套厂" },
  { title: "我要接单", type: "order", desc: "发布产能，承接订单" }
];

export default function HomePage() {
  return (
    <View className="page home-page">
      <View className="hero">
        <Text className="hero-title">织里服装资源撮合</Text>
        <Text className="hero-subtitle">找货、找厂、清库存、接订单</Text>
      </View>

      <View className="action-grid">
        {actions.map((action) => (
          <View
            key={action.type}
            className="action-card"
            onClick={() => Taro.navigateTo({ url: `/pages/resources/index?type=${action.type}` })}
          >
            <Text className="action-title">{action.title}</Text>
            <Text className="action-desc">{action.desc}</Text>
          </View>
        ))}
      </View>

      <View className="publish-bar" onClick={() => Taro.navigateTo({ url: "/pages/publish/index" })}>
        <Text>发布货源/库存/工厂产能</Text>
      </View>
    </View>
  );
}
```

Write `apps/mobile/src/pages/home/index.scss`:

```scss
.hero {
  padding: 32px 28px;
  background: #111827;
  color: #fff;
  border-radius: 8px;
}

.hero-title,
.hero-subtitle,
.action-title,
.action-desc {
  display: block;
}

.hero-title {
  font-size: 36px;
  font-weight: 700;
}

.hero-subtitle {
  margin-top: 10px;
  font-size: 24px;
  color: #d1d5db;
}

.action-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 18px;
  margin-top: 24px;
}

.action-card {
  min-height: 150px;
  padding: 24px;
  background: #fff;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  box-sizing: border-box;
}

.action-title {
  font-size: 30px;
  font-weight: 700;
}

.action-desc {
  margin-top: 12px;
  color: #6b7280;
  font-size: 22px;
  line-height: 1.4;
}

.publish-bar {
  margin-top: 24px;
  padding: 24px;
  color: #fff;
  text-align: center;
  background: #0f766e;
  border-radius: 8px;
}
```

- [ ] **Step 4: Build resource list, detail, and publish pages**

Write `apps/mobile/src/pages/resources/index.config.ts`:

```ts
export default definePageConfig({
  navigationBarTitleText: "资源列表"
});
```

Write `apps/mobile/src/pages/resources/index.tsx`:

```tsx
import { useEffect, useState } from "react";
import Taro, { useRouter } from "@tarojs/taro";
import { Text, View } from "@tarojs/components";
import type { ResourceListItem, ResourceType } from "@wplink/shared";
import { listResources } from "../../services/api";
import "./index.scss";

const typeTitle: Record<string, string> = {
  goods: "找货",
  inventory: "清库存",
  factory: "找厂",
  order: "接订单",
  job: "招聘",
  rental: "出租/转让",
  service: "配套服务"
};

export default function ResourcesPage() {
  const router = useRouter();
  const type = router.params.type as ResourceType | undefined;
  const [items, setItems] = useState<ResourceListItem[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    listResources({ city: "zhili", type })
      .then(setItems)
      .finally(() => setLoading(false));
  }, [type]);

  return (
    <View className="page resources-page">
      <View className="section-head">
        <Text className="section-title">{type ? typeTitle[type] : "织里资源"}</Text>
        <Text className="section-subtitle">优先展示已认证和最近刷新的信息</Text>
      </View>

      {loading && <Text className="empty">加载中</Text>}

      {!loading && items.length === 0 && <Text className="empty">暂无资源，先发布一条</Text>}

      <View className="resource-list">
        {items.map((item) => (
          <View
            key={item.id}
            className="resource-card"
            onClick={() => Taro.navigateTo({ url: `/pages/detail/index?id=${item.id}` })}
          >
            <View className="resource-row">
              <Text className="resource-title">{item.title}</Text>
              {item.isVerified && <Text className="verified">已认证</Text>}
            </View>
            <Text className="resource-meta">{item.category} / {item.district ?? "织里"}</Text>
            <Text className="resource-line">{item.priceText ?? "价格面议"}</Text>
            <Text className="resource-line">{item.quantityText ?? "数量面议"}</Text>
          </View>
        ))}
      </View>
    </View>
  );
}
```

Write `apps/mobile/src/pages/resources/index.scss`:

```scss
.section-title,
.section-subtitle,
.resource-title,
.resource-meta,
.resource-line,
.empty {
  display: block;
}

.section-title {
  font-size: 34px;
  font-weight: 700;
}

.section-subtitle {
  margin-top: 8px;
  color: #6b7280;
  font-size: 22px;
}

.resource-list {
  margin-top: 20px;
}

.resource-card {
  margin-bottom: 16px;
  padding: 22px;
  background: #fff;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
}

.resource-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.resource-title {
  flex: 1;
  font-size: 28px;
  font-weight: 700;
}

.verified {
  padding: 4px 10px;
  color: #0f766e;
  background: #ccfbf1;
  border-radius: 6px;
  font-size: 20px;
}

.resource-meta,
.resource-line,
.empty {
  margin-top: 10px;
  color: #4b5563;
  font-size: 22px;
}
```

Write `apps/mobile/src/pages/detail/index.config.ts`:

```ts
export default definePageConfig({
  navigationBarTitleText: "资源详情"
});
```

Write `apps/mobile/src/pages/detail/index.tsx`:

```tsx
import { useEffect, useState } from "react";
import Taro, { useRouter } from "@tarojs/taro";
import { Button, Text, View } from "@tarojs/components";
import type { ResourceDetail } from "@wplink/shared";
import { getResource, trackContact } from "../../services/api";
import "./index.scss";

export default function DetailPage() {
  const router = useRouter();
  const id = router.params.id;
  const [resource, setResource] = useState<ResourceDetail | null>(null);

  useEffect(() => {
    if (id) {
      getResource(id).then(setResource);
    }
  }, [id]);

  async function callPhone() {
    if (!resource) return;
    await trackContact(resource.id, "phone");
    await Taro.makePhoneCall({ phoneNumber: resource.contact.phone });
  }

  async function copyWechat() {
    if (!resource?.contact.wechat) return;
    await trackContact(resource.id, "wechat");
    await Taro.setClipboardData({ data: resource.contact.wechat });
  }

  if (!resource) {
    return (
      <View className="page detail-page">
        <Text className="empty">加载中</Text>
      </View>
    );
  }

  return (
    <View className="page detail-page">
      <View className="detail-card">
        <View className="title-row">
          <Text className="title">{resource.title}</Text>
          {resource.isVerified && <Text className="verified">已认证</Text>}
        </View>
        <Text className="meta">{resource.category} / {resource.district ?? "织里"}</Text>
        <Text className="desc">{resource.description}</Text>
      </View>

      <View className="detail-card">
        <Text className="block-title">关键信息</Text>
        <Text className="line">价格：{resource.priceText ?? "面议"}</Text>
        <Text className="line">数量：{resource.quantityText ?? "面议"}</Text>
        {Object.entries(resource.attributes).map(([key, value]) => (
          <Text className="line" key={key}>{key}：{String(value)}</Text>
        ))}
      </View>

      <View className="contact-bar">
        <Button className="contact-button" onClick={callPhone}>拨打电话</Button>
        {resource.contact.wechat && (
          <Button className="contact-button secondary" onClick={copyWechat}>复制微信</Button>
        )}
      </View>
    </View>
  );
}
```

Write `apps/mobile/src/pages/detail/index.scss`:

```scss
.detail-card {
  margin-bottom: 16px;
  padding: 24px;
  background: #fff;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
}

.title-row {
  display: flex;
  align-items: flex-start;
  gap: 12px;
}

.title,
.meta,
.desc,
.block-title,
.line,
.empty {
  display: block;
}

.title {
  flex: 1;
  font-size: 32px;
  font-weight: 700;
}

.verified {
  padding: 4px 10px;
  color: #0f766e;
  background: #ccfbf1;
  border-radius: 6px;
  font-size: 20px;
}

.meta,
.line,
.empty {
  margin-top: 10px;
  color: #4b5563;
  font-size: 22px;
}

.desc {
  margin-top: 18px;
  color: #111827;
  font-size: 24px;
  line-height: 1.5;
}

.block-title {
  font-size: 26px;
  font-weight: 700;
}

.contact-bar {
  display: flex;
  gap: 14px;
}

.contact-button {
  flex: 1;
  color: #fff;
  background: #0f766e;
  border-radius: 8px;
}

.contact-button.secondary {
  background: #111827;
}
```

Write `apps/mobile/src/pages/publish/index.config.ts`:

```ts
export default definePageConfig({
  navigationBarTitleText: "发布资源"
});
```

Write `apps/mobile/src/pages/publish/index.tsx`:

```tsx
import { useState } from "react";
import Taro from "@tarojs/taro";
import { Button, Input, Picker, Text, Textarea, View } from "@tarojs/components";
import type { ResourceType } from "@wplink/shared";
import { createResource } from "../../services/api";
import "./index.scss";

const typeOptions: Array<{ label: string; value: ResourceType }> = [
  { label: "找货", value: "goods" },
  { label: "清库存", value: "inventory" },
  { label: "找厂", value: "factory" },
  { label: "接订单", value: "order" }
];

export default function PublishPage() {
  const [typeIndex, setTypeIndex] = useState(1);
  const [title, setTitle] = useState("");
  const [category, setCategory] = useState("童装");
  const [description, setDescription] = useState("");
  const [priceText, setPriceText] = useState("");
  const [quantityText, setQuantityText] = useState("");
  const [contactName, setContactName] = useState("");
  const [contactPhone, setContactPhone] = useState("");
  const [contactWechat, setContactWechat] = useState("");

  async function submit() {
    if (!title || !description || !contactName || !contactPhone) {
      await Taro.showToast({ title: "请补充标题、描述和联系人", icon: "none" });
      return;
    }

    await createResource({
      type: typeOptions[typeIndex].value,
      city: "zhili",
      title,
      category,
      priceText,
      quantityText,
      description,
      contact: {
        name: contactName,
        phone: contactPhone,
        wechat: contactWechat || undefined
      },
      attributes: {
        source: "mobile"
      }
    });

    await Taro.showToast({ title: "已提交审核", icon: "success" });
    setTimeout(() => Taro.navigateBack(), 800);
  }

  return (
    <View className="page publish-page">
      <View className="form-card">
        <Text className="label">资源类型</Text>
        <Picker
          mode="selector"
          range={typeOptions.map((item) => item.label)}
          value={typeIndex}
          onChange={(event) => setTypeIndex(Number(event.detail.value))}
        >
          <View className="picker-value">{typeOptions[typeIndex].label}</View>
        </Picker>

        <Text className="label">标题</Text>
        <Input className="input" value={title} onInput={(event) => setTitle(event.detail.value)} placeholder="例如：女童春款卫衣库存整包清" />

        <Text className="label">品类</Text>
        <Input className="input" value={category} onInput={(event) => setCategory(event.detail.value)} placeholder="童装/加工厂/库存尾货" />

        <Text className="label">价格</Text>
        <Input className="input" value={priceText} onInput={(event) => setPriceText(event.detail.value)} placeholder="例如：打包 18 元/件" />

        <Text className="label">数量/产能</Text>
        <Input className="input" value={quantityText} onInput={(event) => setQuantityText(event.detail.value)} placeholder="例如：3200 件或日产 1200 件" />

        <Text className="label">详细说明</Text>
        <Textarea className="textarea" value={description} onInput={(event) => setDescription(event.detail.value)} placeholder="写清尺码、季节、是否支持看货、是否可直播" />

        <Text className="label">联系人</Text>
        <Input className="input" value={contactName} onInput={(event) => setContactName(event.detail.value)} placeholder="联系人姓名" />

        <Text className="label">电话</Text>
        <Input className="input" value={contactPhone} onInput={(event) => setContactPhone(event.detail.value)} placeholder="手机号" />

        <Text className="label">微信</Text>
        <Input className="input" value={contactWechat} onInput={(event) => setContactWechat(event.detail.value)} placeholder="可选" />

        <Button className="submit-button" onClick={submit}>提交审核</Button>
      </View>
    </View>
  );
}
```

Write `apps/mobile/src/pages/publish/index.scss`:

```scss
.form-card {
  padding: 24px;
  background: #fff;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
}

.label {
  display: block;
  margin-top: 20px;
  color: #374151;
  font-size: 22px;
  font-weight: 700;
}

.label:first-child {
  margin-top: 0;
}

.input,
.textarea,
.picker-value {
  width: 100%;
  margin-top: 10px;
  padding: 18px;
  background: #f9fafb;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  box-sizing: border-box;
  font-size: 24px;
}

.textarea {
  min-height: 160px;
}

.submit-button {
  margin-top: 28px;
  color: #fff;
  background: #0f766e;
  border-radius: 8px;
}
```

- [ ] **Step 5: Run mobile typecheck**

Run:

```bash
npm --workspace apps/mobile run typecheck
```

Expected: PASS.

- [ ] **Step 6: Build WeChat mini-program output**

Run:

```bash
npm --workspace apps/mobile run build:weapp
```

Expected: `apps/mobile/dist` is generated.

- [ ] **Step 7: Commit**

```bash
git add apps/mobile package.json package-lock.json
git commit -m "feat: add zhili mobile mvp"
```

## Task 6: Build Admin Console

**Files:**
- Create: `apps/admin/package.json`
- Create: `apps/admin/index.html`
- Create: `apps/admin/tsconfig.json`
- Create: `apps/admin/vite.config.ts`
- Create: `apps/admin/src/main.tsx`
- Create: `apps/admin/src/App.tsx`
- Create: `apps/admin/src/api.ts`
- Create: `apps/admin/src/styles.css`

- [ ] **Step 1: Create admin package**

Write `apps/admin/package.json`:

```json
{
  "name": "@wplink/admin",
  "version": "0.1.0",
  "private": true,
  "type": "module",
  "scripts": {
    "dev": "vite --host 0.0.0.0 --port 3001",
    "build": "vite build",
    "typecheck": "tsc -p tsconfig.json --noEmit"
  },
  "dependencies": {
    "@vitejs/plugin-react": "^4.3.1",
    "@wplink/shared": "file:../../packages/shared",
    "vite": "^5.4.2",
    "react": "^18.3.1",
    "react-dom": "^18.3.1"
  },
  "devDependencies": {
    "@types/react": "^18.3.4",
    "@types/react-dom": "^18.3.0"
  }
}
```

Write `apps/admin/index.html`:

```html
<div id="root"></div>
<script type="module" src="/src/main.tsx"></script>
```

Write `apps/admin/tsconfig.json`:

```json
{
  "extends": "../../tsconfig.base.json",
  "compilerOptions": {
    "jsx": "react-jsx"
  },
  "include": ["src", "vite.config.ts"]
}
```

Write `apps/admin/vite.config.ts`:

```ts
import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";

export default defineConfig({
  plugins: [react()],
  server: {
    port: 3001
  }
});
```

- [ ] **Step 2: Add admin API client**

Write `apps/admin/src/api.ts`:

```ts
const baseUrl = import.meta.env.VITE_API_BASE_URL ?? "http://127.0.0.1:4000";

export type AdminResource = {
  id: string;
  type: string;
  status: string;
  city: string;
  title: string;
  category: string;
  description: string;
  contactName: string;
  contactPhone: string;
  createdAt: string;
};

export async function listPendingResources() {
  const response = await fetch(`${baseUrl}/admin/resources?status=pending`);
  const data = await response.json();
  return data.items as AdminResource[];
}

export async function reviewResource(id: string, action: "publish" | "reject" | "remove", reason?: string) {
  const response = await fetch(`${baseUrl}/admin/resources/${id}/review`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ action, reason })
  });

  return response.json();
}
```

- [ ] **Step 3: Add admin UI**

Write `apps/admin/src/main.tsx`:

```tsx
import React from "react";
import { createRoot } from "react-dom/client";
import { App } from "./App";
import "./styles.css";

createRoot(document.getElementById("root") as HTMLElement).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>
);
```

Write `apps/admin/src/App.tsx`:

```tsx
import { useEffect, useState } from "react";
import { type AdminResource, listPendingResources, reviewResource } from "./api";

export function App() {
  const [items, setItems] = useState<AdminResource[]>([]);

  async function refresh() {
    setItems(await listPendingResources());
  }

  useEffect(() => {
    void refresh();
  }, []);

  async function handleReview(id: string, action: "publish" | "reject") {
    await reviewResource(id, action, action === "reject" ? "信息不完整，请补充关键字段" : undefined);
    await refresh();
  }

  return (
    <main className="shell">
      <header className="header">
        <h1>衣货通审核台</h1>
        <button onClick={refresh}>刷新</button>
      </header>

      <section className="panel">
        <h2>待审核信息</h2>
        <div className="list">
          {items.map((item) => (
            <article className="row" key={item.id}>
              <div>
                <strong>{item.title}</strong>
                <p>{item.city} / {item.type} / {item.category}</p>
                <p>{item.description}</p>
                <p>{item.contactName} {item.contactPhone}</p>
              </div>
              <div className="actions">
                <button onClick={() => handleReview(item.id, "publish")}>通过</button>
                <button className="secondary" onClick={() => handleReview(item.id, "reject")}>拒绝</button>
              </div>
            </article>
          ))}
          {items.length === 0 && <p className="empty">暂无待审核信息</p>}
        </div>
      </section>
    </main>
  );
}
```

Write `apps/admin/src/styles.css`:

```css
body {
  margin: 0;
  background: #f6f7f9;
  color: #111827;
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
}

.shell {
  max-width: 1120px;
  margin: 0 auto;
  padding: 32px;
}

.header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.panel {
  margin-top: 24px;
  padding: 24px;
  background: #fff;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
}

.row {
  display: grid;
  grid-template-columns: 1fr auto;
  gap: 24px;
  padding: 18px 0;
  border-top: 1px solid #e5e7eb;
}

.row:first-child {
  border-top: 0;
}

.row p {
  margin: 8px 0 0;
  color: #4b5563;
}

.actions {
  display: flex;
  gap: 10px;
  align-items: center;
}

button {
  height: 36px;
  padding: 0 14px;
  border: 0;
  border-radius: 6px;
  background: #0f766e;
  color: #fff;
  cursor: pointer;
}

button.secondary {
  background: #374151;
}

.empty {
  color: #6b7280;
}
```

- [ ] **Step 4: Run admin checks**

Run:

```bash
npm --workspace apps/admin run typecheck
npm --workspace apps/admin run build
```

Expected: both commands exit with code `0`.

- [ ] **Step 5: Commit**

```bash
git add apps/admin package.json package-lock.json
git commit -m "feat: add moderation admin console"
```

## Task 7: End-To-End Local Verification

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Add local runbook**

Write `README.md`:

````md
# 服装产业带资源撮合平台

首发站：织里  
MVP 主线：找货、清库存、找厂、接订单

## Local Setup

```bash
npm install
cp .env.example .env
npm --workspace apps/api run prisma:generate
npm --workspace apps/api run prisma:migrate -- --name init
npm --workspace apps/api run prisma:seed
```

## Run

```bash
npm run dev:api
npm run dev:admin
npm run dev:mobile
```

API: `http://127.0.0.1:4000`  
Admin: `http://127.0.0.1:3001`  
Mini-program output: `apps/mobile/dist`

## Checks

```bash
npm run typecheck
npm run test
npm run build
```

## MVP Acceptance

- Mobile home page shows four primary actions: 找货、清货、找厂、接单.
- Published Zhili seed resources are visible in resource lists.
- New resource submissions enter pending review.
- Admin console can publish or reject pending resources.
- Published resource detail shows contact actions.
- Contact actions create tracking records through the API.
````

- [ ] **Step 2: Run API health check**

Start the API:

```bash
npm run dev:api
```

In another terminal:

```bash
curl http://127.0.0.1:4000/health
```

Expected response:

```json
{"ok":true}
```

- [ ] **Step 3: Verify resource list API**

Run:

```bash
curl "http://127.0.0.1:4000/resources?city=zhili"
```

Expected: JSON response has an `items` array with the seeded inventory and factory records.

- [ ] **Step 4: Verify checks**

Run:

```bash
npm run typecheck
npm run test
npm run build
```

Expected: all commands exit with code `0`.

- [ ] **Step 5: Commit**

```bash
git add README.md
git commit -m "docs: add local mvp runbook"
```

## Self-Review

Spec coverage:

- Product positioning is implemented through the four primary mobile actions.
- Zhili as first station is represented in seed data, city catalog, and default mobile queries.
- Find goods, clear inventory, find factory, and accept orders are represented by resource types.
- Verification is represented by the verification model, route, and admin pending queue.
- Manual review is represented by admin moderation.
- Recruitment, rental, and services are represented as auxiliary resource types.
- Guangzhou and other cities are represented in catalogs and shared city codes, without adding city-specific UI.

Deferred by design:

- Escrow, checkout, chat, commission, and city-operator permissions remain outside this MVP because the PRD explicitly places them beyond the first validation stage.

Plan quality checks:

- Each implementation task has exact file paths.
- Each code-writing step includes concrete file contents or concrete code blocks.
- Verification commands include expected results.
- The plan keeps the first release focused on a testable Zhili vertical slice.

## Execution Choice

Plan complete and saved to `docs/superpowers/plans/2026-06-26-apparel-platform-mvp.md`. Two execution options:

1. **Subagent-Driven (recommended)** - Dispatch a fresh subagent per task, review between tasks, faster iteration.
2. **Inline Execution** - Execute tasks in this session using executing-plans, batch execution with checkpoints.

Choose one approach before implementation starts.
