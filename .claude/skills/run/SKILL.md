---
name: run
description: "Trigger: run web, start dev server, preview landing, screenshot page, verify Astro app. Launch the /web Astro dev server and screenshot a page with Playwright."
license: Apache-2.0
metadata:
  author: gentleman-programming
  version: "1.0"
---

## Activation Contract

Use when asked to run, preview, or screenshot the `/web` Astro app during development — not for `astro build` alone.

## Hard Rules

- `chromium-cli` is NOT installed in this environment. Do not attempt it. Use Playwright directly (Execution Steps).
- The Google Fonts `@import` in `web/src/styles/global.css` MUST stay above `@import "tailwindcss";`. Tailwind v4 inlines its own content at that line, so an `@import` placed after it ends up behind non-import rules and Lightning CSS rejects it with an `@import rules must precede all rules` warning.

## Execution Steps

1. Start the dev server in the background and poll until it actually serves — never `sleep` a fixed guess:
   ```bash
   cd web
   (npm run dev -- --port 4321 > /tmp/god-edu-web-dev.log 2>&1 &)
   timeout 30 bash -c 'until curl -sf http://localhost:4321 >/dev/null; do sleep 1; done'
   ```
2. Drive it with Playwright — install it once per session in the scratchpad dir (never inside `web/`, it is not a project dependency):
   ```bash
   cd <scratchpad>
   npm init -y >/dev/null && npm install playwright >/dev/null
   npx --yes playwright install chromium --with-deps   # first run only, downloads the browser
   ```
   Then run a Node ESM script (`node script.mjs`) from that same directory:
   ```js
   import { chromium } from 'playwright';
   const browser = await chromium.launch({ args: ['--no-sandbox'] });
   const page = await browser.newPage({ viewport: { width: 1280, height: 800 } });
   const errors = [];
   page.on('pageerror', (e) => errors.push(String(e)));
   page.on('console', (m) => { if (m.type() === 'error') errors.push(m.text()); });
   await page.goto('http://localhost:4321', { waitUntil: 'networkidle' });
   await page.waitForSelector('text=<something unique to the page under test>');
   await page.screenshot({ path: '<scratchpad>/screenshot.png', fullPage: true });
   console.log('errors:', JSON.stringify(errors));
   await browser.close();
   ```
3. Stop the server: `lsof -ti:4321 -sTCP:LISTEN | xargs -r kill`. Avoid broad `pkill -f` — it can match the agent's own shell.
4. Read the screenshot with the Read tool and actually look at it before declaring success. Check `errors` is empty — a page can render its shell while a data fetch fails silently.

## Output Contract

Report: whether the dev server started cleanly, the screenshot path, and any console/page errors found.

## References

- Bundled `run` skill (generic launch patterns) — `examples/playwright.md` for the browser-driven pattern this was adapted from.
