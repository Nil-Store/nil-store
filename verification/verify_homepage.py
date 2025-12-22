
import asyncio
from playwright.async_api import async_playwright

async def run():
    async with async_playwright() as p:
        browser = await p.chromium.launch()
        page = await browser.new_page()
        # Set viewport to standard desktop
        await page.set_viewport_size({"width": 1280, "height": 800})

        # Navigate to localhost
        try:
            await page.goto("http://localhost:5173", timeout=30000)
            # Wait for content to load (logo, text)
            await page.wait_for_selector('h1')
            await page.wait_for_timeout(2000) # Wait for animations

            # Take screenshot
            await page.screenshot(path="/home/jules/verification/homepage_word_wrap.png", full_page=True)
            print("Screenshot saved to /home/jules/verification/homepage_word_wrap.png")

        except Exception as e:
            print(f"Error: {e}")

        await browser.close()

if __name__ == "__main__":
    asyncio.run(run())
