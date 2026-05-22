import { expect, test } from "@playwright/test"

test.describe("Dashboard", () => {
  test("renders the main sections", async ({ page }) => {
    await page.goto("/")
    await expect(page.getByRole("heading", { name: /fraud control/i })).toBeVisible()
    await expect(page.getByText("Rule contribution")).toBeVisible()
    await expect(page.getByText("Score distribution")).toBeVisible()
    await expect(page.getByText("Recent transactions")).toBeVisible()
    await expect(page.getByTestId("notifications-panel")).toBeVisible()
  })

  test("connects to the websocket and shows the hello notification", async ({
    page,
  }) => {
    await page.goto("/")
    await expect(page.getByText(/ws:\s*open/i)).toBeVisible({ timeout: 5_000 })
    await expect(
      page.getByTestId("notifications-panel").getByText("Connected"),
    ).toBeVisible({ timeout: 5_000 })
  })

  test("send test webhook button pushes a notification", async ({ page }) => {
    await page.goto("/")
    await expect(page.getByText(/ws:\s*open/i)).toBeVisible({ timeout: 5_000 })

    const panel = page.getByTestId("notifications-panel")
    await page.getByTestId("send-test-webhook").click()

    // Wait for at least one webhook.test message to land alongside the hello.
    await expect(panel).toContainText(/High-risk|Velocity|Chargeback|Daily report/, {
      timeout: 5_000,
    })
  })
})
