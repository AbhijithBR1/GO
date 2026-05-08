package main

import (
	"fmt"
	"log"
	"time"

	"github.com/playwright-community/playwright-go"
)

func main() {
	// 1. Start Playwright
	pw, err := playwright.Run()
	if err != nil {
		log.Fatalf("Could not start playwright: %v", err)
	}

	// 2. Launch with your ACTUAL profile
	// Change the path below to your actual path from Phase 1
	userDataDir := "C:\\Users\\TLUser\\Desktop\\Abhijith"

	browser, err := pw.Chromium.LaunchPersistentContext(userDataDir, playwright.BrowserTypeLaunchPersistentContextOptions{
		Headless: playwright.Bool(true),
		Channel:  playwright.String("chrome"),
	})
	if err != nil {
		log.Fatalf("Could not launch browser: %v", err)
	}
	defer browser.Close()

	page, _ := browser.NewPage()
	
	// 3. Go to Teams
	fmt.Println("Opening Teams...")
	page.Goto("https://teams.live.com/v2/")

	// 4. Click the Lunch Group
	// This looks for the word "Own" in the sidebar.
	// Long timeout so on the first run you have time to log in to Teams manually;
	// after that, the session is saved in the user-data dir and future runs are instant.
	lunchGroup := page.Locator("span:has-text('Own')").First()
	if err := lunchGroup.Click(playwright.LocatorClickOptions{
		Timeout: playwright.Float(120000),
	}); err != nil {
		log.Fatalf("Could not find Lunch group: %v", err)
	}

	// 5. Find the latest Google Form Link
	fmt.Println("Searching for the latest link...")
	selector := "a[href*='docs.google.com']"
	page.WaitForSelector(selector)
	
	links, _ := page.QuerySelectorAll(selector)
	lastLink, _ := links[len(links)-1].GetAttribute("href")
	fmt.Println("Found Link:", lastLink)

	// 6. Fill the Form
	fillAndSubmit(browser, lastLink)
}

func fillAndSubmit(browser playwright.BrowserContext, link string) {
	fmt.Println("Opening form:", link)
	page, _ := browser.NewPage()
	if _, err := page.Goto(link); err != nil {
		log.Fatalf("Could not open form: %v", err)
	}
	fmt.Println("Form page loaded")

	// Wait for name field and fill it
	fmt.Println("Waiting for name field...")
	if _, err := page.WaitForSelector("input[type='text']"); err != nil {
		log.Fatalf("Name field never appeared: %v", err)
	}
	fmt.Println("Filling name: Abhijith B R")
	if err := page.Fill("input[type='text']", "Abhijith B R"); err != nil {
		log.Fatalf("Could not fill name: %v", err)
	}

	// Click the first radio button option
	fmt.Println("Selecting first radio option...")
	if err := page.Click("div[role='radio']"); err != nil {
		log.Fatalf("Could not click radio: %v", err)
	}

	// Click Submit
	fmt.Println("Clicking Submit...")
	if err := page.Click("span:has-text('Submit')"); err != nil {
		log.Fatalf("Could not click Submit: %v", err)
	}

	// Wait for Google's confirmation page so the POST actually completes
	// before the browser closes. Without this, the click fires but the
	// network request can be cancelled by browser.Close().
	fmt.Println("Waiting for Google to confirm submission...")
	if err := page.WaitForURL("**/formResponse*", playwright.PageWaitForURLOptions{
		Timeout: playwright.Float(15000),
	}); err != nil {
		log.Fatalf("Submission not confirmed (form did not redirect): %v", err)
	}
	fmt.Println("Confirmed URL:", page.URL())
	fmt.Println("Success! Form submitted at:", time.Now().Format("15:04"))
}