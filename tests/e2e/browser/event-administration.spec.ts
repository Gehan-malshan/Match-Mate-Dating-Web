import { expect, test } from '@playwright/test'
import path from 'node:path'

const adminEmail = process.env.MATCHMATE_E2E_ADMIN_EMAIL ?? 'admin@example.test'
const adminPassword = process.env.MATCHMATE_E2E_ADMIN_PASSWORD ?? 'MatchMateDev123!'

test.beforeAll(async ({ request }) => {
  for (const url of ['http://localhost:8080/health/live', 'http://localhost:8082/health/ready', 'http://localhost:8085/health/live']) {
    const response = await request.get(url)
    expect(response, `${url} must be ready; start the backend with docker compose up --build -d`).toBeOK()
  }
})

test('administrator completes the event lifecycle and opens registrations', async ({ page }) => {
  const eventName = `Browser Safety Review ${Date.now()}`

  await test.step('sign in and create a draft with an image', async () => {
    await page.goto('/login')
    await page.getByLabel('Email address').fill(adminEmail)
    await page.locator('input[autocomplete="current-password"]').fill(adminPassword)
    await page.getByRole('button', { name: /Continue your journey/i }).click()
    await expect(page).toHaveURL(/\/admin$/)

    await page.getByRole('button', { name: 'Event manager' }).click()
    await page.getByRole('button', { name: /Create new event/i }).click()
    await page.getByLabel('Event name').fill(eventName)
    await page.getByLabel('Public description').fill('A browser-tested event using fictional data and the approved safety workflow.')
    await page.getByLabel(/Exact venue/).fill('Test venue — not a real address')
    await page.getByLabel(/Upload event photo/).setInputFiles(path.resolve('frontend/apps/web/public/images/matchmate-event.png'))
    await expect(page.getByAltText('Preview of selected event artwork')).toBeVisible()
    await page.getByRole('button', { name: 'Save private draft' }).click()

    const card = page.locator('article.event-card').filter({ hasText: eventName })
    await expect(card).toBeVisible()
    await expect(card).toContainText('DRAFT')
    await expect(card).toContainText('Attached')
  })

  await test.step('require rights, moderation, and safety approval before publication', async () => {
    const card = page.locator('article.event-card').filter({ hasText: eventName })
    await card.getByRole('button', { name: 'Publish' }).click()
    await expect(page.getByRole('heading', { name: `Approve ${eventName}` })).toBeVisible()
    await page.getByLabel(/owns or has documented permission/i).check()
    await page.getByLabel(/reviewed the image for minors/i).check()
    await page.getByLabel(/confirmed the venue contact/i).check()
    await page.getByRole('button', { name: 'Approve and publish' }).click()
    await expect(card).toContainText('PUBLISHED')
  })

  await test.step('cancel, recoverably delete, and restore the event', async () => {
    const card = page.locator('article.event-card').filter({ hasText: eventName })
    page.once('dialog', (dialog) => dialog.accept())
    await card.getByRole('button', { name: 'Cancel' }).click()
    await expect(card).toContainText('CANCELLED')

    page.once('dialog', (dialog) => dialog.accept())
    await card.getByRole('button', { name: 'Delete' }).click()
    await expect(card).toHaveCount(0)

    await page.getByRole('button', { name: 'Show deleted' }).click()
    const deletedCard = page.locator('article.event-card').filter({ hasText: eventName })
    await expect(deletedCard).toBeVisible()
    page.once('dialog', (dialog) => dialog.accept())
    await deletedCard.getByRole('button', { name: 'Restore' }).click()
    await expect(deletedCard).toHaveCount(0)

    await page.getByRole('button', { name: 'Show active' }).click()
    await expect(page.locator('article.event-card').filter({ hasText: eventName })).toBeVisible()
  })

  await test.step('open the event-scoped registration view', async () => {
    const card = page.locator('article.event-card').filter({ hasText: eventName })
    await card.getByRole('button', { name: 'View registrations' }).click()
    await expect(page.getByRole('heading', { name: 'Event registrations.' })).toBeVisible()
    await expect(page.getByRole('heading', { name: eventName })).toBeVisible()
    await expect(page.locator('#registration-event-filter')).toHaveValue(/.+/)
    await expect(page.getByText('No bookings have been created for this selection yet.')).toBeVisible()
  })
})
