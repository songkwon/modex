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
  await page.route("http://localhost:8671/api/auth/me", async (route) => {
    await route.fulfill({ status: 401, contentType: "application/json", body: JSON.stringify({ error: { code: "unauthorized" } }) });
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

test.beforeEach(async ({ page }) => {
  await page.addInitScript(() => {
    window.localStorage.setItem("modex_locale", "zh-CN");
  });
  await mockBackend(page);
});

test("home page renders and switches locale", async ({ page }) => {
  await page.goto("/");

  await expect(page.getByRole("heading", { name: "文档中心" })).toBeVisible();
  await expect(page.getByRole("button", { name: "登录" })).toBeVisible();

  await page.getByTestId("locale-select").selectOption("en-US");
  await expect(page.getByRole("heading", { name: "Documentation Hub" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Log in" })).toBeVisible();
});

test("mock login exposes the admin console entry", async ({ page }) => {
  await page.goto("/");

  await page.getByRole("button", { name: "登录" }).click();
  await page.getByRole("button", { name: /Dev User/ }).click();

  await expect(page.getByRole("link", { name: /管理控制台/ })).toBeVisible();
});
