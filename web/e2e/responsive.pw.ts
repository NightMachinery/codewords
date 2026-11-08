import { expect, test, type Page } from '@playwright/test';

async function expectControlsFit(page: Page, selector: string) {
  const problems = await page.locator(selector).evaluate((root) => {
    const issues: string[] = [];
    const box = root.getBoundingClientRect();
    const controls = [...root.querySelectorAll('button, input:not([type="file"]), select')]
      .filter((element) => element.getBoundingClientRect().width > 0);
    for (const control of controls) {
      const rect = control.getBoundingClientRect();
      const label = control.getAttribute('aria-label') || control.getAttribute('placeholder') || control.textContent?.trim();
      if (rect.left < box.left - 1 || rect.right > box.right + 1 || rect.right > innerWidth + 1) {
        issues.push(`Clipped control: ${label} (${rect.left}, ${rect.right}, panel ${box.right})`);
      }
      for (let ancestor = control.parentElement; ancestor && ancestor !== root; ancestor = ancestor.parentElement) {
        if (getComputedStyle(ancestor).overflowX !== 'visible') {
          const clip = ancestor.getBoundingClientRect();
          if (rect.left < clip.left - 1 || rect.right > clip.right + 1) issues.push(`Clipped by ancestor: ${label}`);
        }
      }
      if (control instanceof HTMLButtonElement && !control.disabled) {
        const hit = document.elementFromPoint(rect.left + rect.width / 2, rect.top + rect.height / 2);
        if (rect.top >= 0 && rect.bottom <= innerHeight && hit && !control.contains(hit)) issues.push(`Obscured button: ${label}`);
      }
    }
    for (let i = 0; i < controls.length; i++) {
      for (const other of controls.slice(i + 1)) {
        const a = controls[i].getBoundingClientRect();
        const b = other.getBoundingClientRect();
        if (Math.min(a.right, b.right) - Math.max(a.left, b.left) > 1 && Math.min(a.bottom, b.bottom) - Math.max(a.top, b.top) > 1) {
          issues.push(`Overlapping controls: ${controls[i].getAttribute('aria-label')} / ${other.getAttribute('aria-label')}`);
        }
      }
    }
    return issues;
  });
  expect(problems).toEqual([]);
}

for (const width of [320, 360, 393, 430, 768, 1024, 1440]) {
  for (const role of ['spy', 'guesser']) {
    test(`footer fits ${width}px ${role}`, async ({ page }) => {
      await page.setViewportSize({ width, height: width === 768 ? 393 : 900 });
      await page.goto(`/e2e/responsive.html?role=${role}`);
      await expect(page.getByRole('button', { name: 'Collapse bottom controls' })).toBeVisible();
      await expectControlsFit(page, '#bottom-sticky-panel');
      if (role === 'spy') {
        const input = page.getByPlaceholder('One-word clue');
        await input.fill('Test');
        await page.locator('#bottom-sticky-panel select').selectOption('2');
        const submit = page.getByRole('button', { name: 'Submit clue' });
        const inputBox = await input.boundingBox();
        const submitBox = await submit.boundingBox();
        expect(Math.abs(inputBox!.y - submitBox!.y)).toBeLessThan(3);
        await submit.click();
        await expect(page.getByRole('status')).toHaveText('Clue submitted');
      } else {
        await page.getByRole('button', { name: 'Pass turn' }).click();
        await expect(page.getByRole('status')).toHaveText('Turn passed');
      }
      await page.getByRole('button', { name: 'Show previous board' }).click();
      await expect(page.getByRole('button', { name: 'Show previous board' })).toHaveAttribute('aria-pressed', 'true');
      await page.getByRole('button', { name: 'Collapse bottom controls' }).click();
      await page.getByRole('button', { name: 'Expand bottom controls' }).click();
      await expectControlsFit(page, '#bottom-sticky-panel');
    });
  }

  test(`settings fit ${width}px`, async ({ page }) => {
    await page.setViewportSize({ width, height: 900 });
    await page.goto('/e2e/responsive.html?surface=settings');
    await expect(page.getByText('Wordpacks', { exact: true })).toBeVisible();
    await expectControlsFit(page, '#settings');
    await page.locator('select').last().selectOption('long');
    await page.getByRole('button', { name: 'Add', exact: true }).click();
    await expect(page.getByText('2 selected', { exact: false })).toBeVisible();
    await expectControlsFit(page, '#settings');
    await page.getByRole('button', { name: 'Remove', exact: true }).last().click();
    await expect(page.getByText('1 selected', { exact: false })).toBeVisible();
  });
}

for (const mode of ['polarity', 'monality']) {
  for (const role of ['spy', 'guesser']) {
    test(`${mode} footer fits 320px ${role} with a current clue`, async ({ page }) => {
      await page.setViewportSize({ width: 320, height: 720 });
      await page.goto(`/e2e/responsive.html?mode=${mode}&role=${role}&clue=1`);
      await expect(page.getByRole('button', { name: 'Collapse bottom controls' })).toBeVisible();
      await expectControlsFit(page, '#bottom-sticky-panel');
      await page.getByRole('button', { name: 'Chat', exact: true }).click();
      await expect(page.getByRole('status')).toHaveText('chat');
    });
  }
}
