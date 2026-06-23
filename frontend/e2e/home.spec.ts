import { expect, test, type Page } from "@playwright/test";

async function mockBackend(page: Page) {
  await page.route("http://localhost:8671/api/config", async (route) => {
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        oidc_login_enabled: true,
        login_url: "http://localhost:8671/api/auth/login",
        frontend_base_url: "http://127.0.0.1:3456",
        auto_login: false,
        plugins: {}
      })
    });
  });
  // Logged out by default; loginAs() overrides this for authenticated cases.
  await page.route("http://localhost:8671/api/auth/me*", async (route) => {
    await route.fulfill({ contentType: "application/json", body: JSON.stringify(null) });
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

// loginAs simulates an established session by serving an authenticated
// /api/auth/me. Registered after mockBackend so it takes route precedence.
async function loginAs(page: Page) {
  await page.route("http://localhost:8671/api/auth/me*", async (route) => {
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        id: "dev",
        username: "dev",
        display_name: "Dev User",
        email: "dev@example.com",
        department: "Docs",
        groups: [],
        roles: ["admin"],
        is_super_admin: true
      })
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

test("home page renders and shows the login entry when logged out", async ({ page }) => {
  const pageErrors: string[] = [];
  page.on("pageerror", (error) => pageErrors.push(error.message));

  await page.goto("/");

  await expect(page.getByRole("heading", { name: "文档中心" })).toBeVisible();
  await expect(page.getByRole("button", { name: "登录" })).toBeVisible();
  expect(pageErrors).toEqual([]);
});

test("authenticated user can switch locale from the user menu", async ({ page }) => {
  await loginAs(page);
  await page.goto("/");

  // Starts in zh-CN (seeded via localStorage in beforeEach).
  await expect(page.locator("html")).toHaveAttribute("lang", "zh-CN");

  await page.getByRole("button", { name: /Dev User/ }).click();
  await page.getByRole("button", { name: "English" }).click();

  // The switch takes effect live: html lang flips and t()-rendered UI re-renders.
  await expect(page.locator("html")).toHaveAttribute("lang", "en-US");
  await expect(page.getByRole("heading", { name: "Documentation Hub" })).toBeVisible();
});

test("authenticated admin sees the admin console entry", async ({ page }) => {
  await loginAs(page);
  await page.goto("/");

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
  await expect(page.getByRole("link", { name: "查看项目指南" })).toHaveCount(0);
});
