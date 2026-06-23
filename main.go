package main

import (
	"fmt"
	"log"
	"syscall"
	"time"
	"unsafe"

	"github.com/playwright-community/playwright-go"
)

// askYesNo shows a native Windows Yes/No dialog and returns true for Yes.
// Uses user32.dll MessageBox so it works even when launched by Task Scheduler
// (as long as the task runs in the interactive, logged-on user session).
func askYesNo(caption, text string) bool {
	const (
		MB_YESNO       = 0x00000004
		MB_ICONQUESTION = 0x00000020
		MB_SYSTEMMODAL  = 0x00001000 // keep the dialog on top of other windows
		IDYES           = 6
	)
	user32 := syscall.NewLazyDLL("user32.dll")
	messageBox := user32.NewProc("MessageBoxW")

	textPtr, _ := syscall.UTF16PtrFromString(text)
	captionPtr, _ := syscall.UTF16PtrFromString(caption)

	ret, _, _ := messageBox.Call(
		0,
		uintptr(unsafe.Pointer(textPtr)),
		uintptr(unsafe.Pointer(captionPtr)),
		uintptr(MB_YESNO|MB_ICONQUESTION|MB_SYSTEMMODAL),
	)
	return int(ret) == IDYES
}

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
		Headless: playwright.Bool(false),
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
	lunchGroup := page.Locator("span:has-text('Lunch')").First()
	if err := lunchGroup.Click(playwright.LocatorClickOptions{
		Timeout: playwright.Float(120000),
	}); err != nil {
		log.Fatalf("Could not find Lunch group: %v", err)
	}

	// 5. Find the latest Google Form Link
	fmt.Println("Searching for the latest link...")
	// Match both the short links (forms.gle) and full Google Forms URLs.
	selector := "a[href*='forms.gle'], a[href*='docs.google.com']"
	if _, err := page.WaitForSelector(selector, playwright.PageWaitForSelectorOptions{
		Timeout: playwright.Float(30000),
	}); err != nil {
		log.Fatalf("No Google Form link appeared in the chat within 30s: %v", err)
	}

	links, err := page.QuerySelectorAll(selector)
	if err != nil {
		log.Fatalf("Could not query for links: %v", err)
	}
	if len(links) == 0 {
		log.Fatalf("Selector matched no links (0 found) — the form link may be in an iframe or not yet loaded")
	}
	lastLink, _ := links[len(links)-1].GetAttribute("href")
	fmt.Printf("Found %d link(s). Using latest: %s\n", len(links), lastLink)

	// 6. Ask whether to go for lunch today (native popup)
	goingForLunch := askYesNo("Lunch", "Going for lunch today?")

	// 7. Fill the Form
	fillAndSubmit(browser, lastLink, goingForLunch)
}

func fillAndSubmit(browser playwright.BrowserContext, link string, goingForLunch bool) {
	fmt.Println("Opening form:", link)
	page, _ := browser.NewPage()
	if _, err := page.Goto(link, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); err != nil {
		log.Fatalf("Could not open form: %v", err)
	}
	fmt.Println("Form page loaded. URL:", page.URL())

	// Wait for an actual form question card to render. If this times out, dump
	// the page URL + title so we can see if Google redirected us somewhere else
	// (e.g. an account chooser or consent screen) instead of the form.
	fmt.Println("Waiting for form fields...")
	if _, err := page.WaitForSelector("div[role='listitem']", playwright.PageWaitForSelectorOptions{
		Timeout: playwright.Float(30000),
	}); err != nil {
		title, _ := page.Title()
		log.Fatalf("Form questions never appeared.\n  URL:   %s\n  Title: %s\n  Error: %v", page.URL(), title, err)
	}

	// Fill a field by locating the question card (listitem) whose text contains
	// the given label, then filling the textbox inside it. This stays correct
	// regardless of question order or what aria-label Google puts on the input.
	fillField := func(label, value string) {
		fmt.Printf("Filling %s: %s\n", label, value)
		field := page.Locator("div[role='listitem']").
			Filter(playwright.LocatorFilterOptions{HasText: label}).
			GetByRole("textbox")
		if err := field.Fill(value, playwright.LocatorFillOptions{
			Timeout: playwright.Float(15000),
		}); err != nil {
			log.Fatalf("Could not fill %s: %v", label, err)
		}
	}

	fillField("Email", "abhadran@thoughtlinedigital.com")
	fillField("Name", "Abhijith B R")

	// Select Yes/No by label based on the popup answer.
	choice := "Yes"
	if !goingForLunch {
		choice = "No"
	}
	fmt.Printf("Selecting radio option %q (goingForLunch=%v)...\n", choice, goingForLunch)
	if err := page.GetByRole("radio", playwright.PageGetByRoleOptions{
		Name: choice,
	}).Click(); err != nil {
		log.Fatalf("Could not click radio %q: %v", choice, err)
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