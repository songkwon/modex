import { expect, test, type Page } from "@playwright/test";

async function mockBackend(page: Page) {
  await page.route("http://localhost:8671/api/config", async (route) => {
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        auth_mode: "mock",
        oidc_login_enabled: false,
        login_url: "http://localhost:8671/api/auth/login",
        frontend_base_url: "http://127.0.0.1:3456",
        auto_login: false,
        plugins: {}
      })
    });
  });
  await page.route("http://localhost:8671/api/auth/me*", async (route) => {
    await route.fulfill({ contentType: "application/json", body: JSON.stringify(null) });
  });
  await page.route("http://localhost:8671/api/auth/mock-login", async (route) => {
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        user: {
          id: "dev",
          username: "dev",
          display_name: "Dev User",
          email: "dev@example.com",
          department: "Docs",
          groups: [],
          roles: ["admin"],
          is_super_admin: true
        }
      })
    });
  });
  await page.route("http://localhost:8671/api/categories/tree", async (route) => {
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify([
        {
          id: "engineering",
          key: "engineering",
          name: "Engineering",
          description: "Engineering docs",
          icon: "book",
          sort_order: 1,
          status: "active",
          children: []
        }
      ])
    });
  });
  await page.route("http://localhost:8671/api/modules", async (route) => {
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify([
        {
          id: "m-demo",
          module_key: "DemoModule",
          name: "Demo Module",
          description: "Demo documentation",
          default_version: "latest",
          status: "active",
          category_ids: ["engineering"],
          category_path: "Engineering",
          updated_at: new Date().toISOString()
        }
      ])
    });
  });
}

async function mockEmptyDocs(page: Page) {
  await page.route("http://localhost:8671/api/categories/tree", async (route) => {
    await route.fulfill({ contentType: "application/json", body: JSON.stringify(null) });
  });
  await page.route("http://localhost:8671/api/modules", async (route) => {
    await route.fulfill({ contentType: "application/json", body: JSON.stringify(null) });
  });
}

test.beforeEach(async ({ page }) => {
  await page.addInitScript(() => {
    window.localStorage.setItem("modex_locale", "zh-CN");
  });
  await mockBackend(page);
});

test("home page renders and switches locale from the user menu", async ({ page }) => {
  const pageErrors: string[] = [];
  page.on("pageerror", (error) => pageErrors.push(error.message));

  await page.goto("/");

  await expect(page.getByRole("heading", { name: "文档中心" })).toBeVisible();
  await expect(page.getByRole("button", { name: "登录" })).toBeVisible();
  expect(pageErrors).toEqual([]);

  await page.getByRole("button", { name: "登录" }).click();
  await page.getByRole("button", { name: /Dev User/ }).click();
  await page.getByRole("button", { name: "English" }).click();
  await expect(page.getByRole("heading", { name: "Documentation Hub" })).toBeVisible();
  await expect(page.getByRole("link", { name: /Project guide/ })).toBeVisible();
});

test("mock login exposes the admin console entry", async ({ page }) => {
  await page.goto("/");

  await page.getByRole("button", { name: "登录" }).click();
  await page.getByRole("button", { name: /Dev User/ }).click();

  await expect(page.getByRole("link", { name: /项目指南/ })).toBeVisible();
  await expect(page.getByRole("link", { name: /管理控制台/ })).toBeVisible();
});

test("home page explains the empty documentation state", async ({ page }) => {
  await mockEmptyDocs(page);
  await page.goto("/");

  await expect(page.getByText("还没有接入文档")).toBeVisible();
  await expect(page.getByText("当前空间暂无分类或文档源")).toBeVisible();
  await expect(page.getByRole("link", { name: "接入文档源" })).toHaveAttribute("href", "/admin/modules");
  await expect(page.getByRole("link", { name: "查看项目指南" })).toHaveAttribute("href", "/me/guide");
});
