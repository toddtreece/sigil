import { test, expect } from './fixtures';
import { ROUTES } from '../src/constants';

test.describe('navigating sigil app', () => {
  test('conversations page should render', async ({ gotoPage, page }) => {
    await gotoPage(`/${ROUTES.Conversations}`);
    await expect(page.getByRole('heading', { name: 'Conversations' })).toBeVisible();
  });

  test('completions page should render', async ({ gotoPage, page }) => {
    await gotoPage(`/${ROUTES.Completions}`);
    await expect(page.getByRole('heading', { name: 'Completions' })).toBeVisible();
  });

  test('traces page should render', async ({ gotoPage, page }) => {
    await gotoPage(`/${ROUTES.Traces}`);
    await expect(page.getByRole('heading', { name: 'Traces' })).toBeVisible();
  });

  test('settings page should render', async ({ gotoPage, page }) => {
    await gotoPage(`/${ROUTES.Settings}`);
    await expect(page.getByRole('heading', { name: 'Settings' })).toBeVisible();
  });
});
