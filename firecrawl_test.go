package firecrawl

import (
	"log"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var API_URL string
var TEST_API_KEY string

func ptr[T any](v T) *T {
	return &v
}

func init() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}
	API_URL = os.Getenv("API_URL")
	TEST_API_KEY = os.Getenv("TEST_API_KEY")
}

func TestNoAPIKey(t *testing.T) {
	_, err := NewFirecrawlApp("", API_URL)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no API key provided")
}

func TestScrapeURLInvalidAPIKey(t *testing.T) {
	app, err := NewFirecrawlApp("invalid_api_key", API_URL)
	require.NoError(t, err)

	_, err = app.ScrapeURL("https://firecrawl.dev", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Unexpected error during scrape URL: Status code 401. Unauthorized: Invalid token")
}

func TestBlocklistedURL(t *testing.T) {
	app, err := NewFirecrawlApp(TEST_API_KEY, API_URL)
	require.NoError(t, err)

	_, err = app.ScrapeURL("https://facebook.com/fake-test", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Status code 403")
}

func TestScrapeURLE2E(t *testing.T) {
	app, err := NewFirecrawlApp(TEST_API_KEY, API_URL)
	require.NoError(t, err)

	response, err := app.ScrapeURL("https://www.scrapethissite.com", nil)
	require.NoError(t, err)
	assert.NotNil(t, response)

	assert.Contains(t, response.Markdown, "# Scrape This Site")
	assert.NotEqual(t, response.Markdown, "")
	assert.NotNil(t, response.Metadata)
	assert.Equal(t, response.HTML, "")
}

func TestSuccessfulResponseWithValidAPIKeyAndIncludeHTML(t *testing.T) {
	app, err := NewFirecrawlApp(TEST_API_KEY, API_URL)
	require.NoError(t, err)

	params := ScrapeParams{
		Formats:         []string{"markdown", "html", "rawHtml", "screenshot", "links"},
		Headers:         ptr(map[string]string{"x-key": "test"}),
		IncludeTags:     []string{"h1"},
		ExcludeTags:     []string{"h2"},
		OnlyMainContent: ptr(true),
		Timeout:         ptr(30000),
		WaitFor:         ptr(1000),
	}

	response, err := app.ScrapeURL("https://www.scrapethissite.com", &params)
	require.NoError(t, err)
	assert.NotNil(t, response)

	assert.Contains(t, response.Markdown, "# Scrape This Site")
	assert.Contains(t, response.HTML, "<h1")
	assert.Contains(t, response.RawHTML, "<h1")
	assert.NotNil(t, response.Screenshot)
	assert.NotEmpty(t, response.Screenshot)
	assert.Contains(t, response.Screenshot, "https://")
	assert.NotNil(t, response.Links)
	assert.NotNil(t, response.Metadata)
}

func TestSuccessfulResponseForValidScrapeWithPDFFile(t *testing.T) {
	app, err := NewFirecrawlApp(TEST_API_KEY, API_URL)
	require.NoError(t, err)

	response, err := app.ScrapeURL("https://arxiv.org/pdf/astro-ph/9301001.pdf", nil)
	require.NoError(t, err)
	assert.NotNil(t, response)

	assert.Contains(t, response.Markdown, "We present spectrophotometric observations of the Broad Line Radio Galaxy")
	assert.NotNil(t, response.Metadata)
}

func TestSuccessfulResponseForValidScrapeWithPDFFileWithoutExplicitExtension(t *testing.T) {
	app, err := NewFirecrawlApp(TEST_API_KEY, API_URL)
	require.NoError(t, err)

	response, err := app.ScrapeURL("https://arxiv.org/pdf/astro-ph/9301001", nil)
	require.NoError(t, err)
	assert.NotNil(t, response)

	assert.Contains(t, response.Markdown, "We present spectrophotometric observations of the Broad Line Radio Galaxy")
	assert.NotNil(t, response.Metadata)
}

func TestCrawlURLInvalidAPIKey(t *testing.T) {
	app, err := NewFirecrawlApp("invalid_api_key", API_URL)
	require.NoError(t, err)

	_, err = app.CrawlURL("https://firecrawl.dev", nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Unexpected error during start crawl job: Status code 401. Unauthorized: Invalid token")
}

func TestShouldReturnErrorForBlocklistedURL(t *testing.T) {
	app, err := NewFirecrawlApp(TEST_API_KEY, API_URL)
	require.NoError(t, err)

	_, err = app.CrawlURL("https://twitter.com/fake-test", nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Status code 403")
}

func TestCrawlURLE2E(t *testing.T) {
	app, err := NewFirecrawlApp(TEST_API_KEY, API_URL)
	require.NoError(t, err)

	response, err := app.CrawlURL("https://www.scrapethissite.com", nil, nil)
	require.NoError(t, err)
	assert.NotNil(t, response)

	assert.Greater(t, response.Total, 0)
	assert.Greater(t, response.Completed, 0)
	assert.Greater(t, response.CreditsUsed, 0)
	assert.NotEmpty(t, response.ExpiresAt)
	assert.Equal(t, response.Status, "completed")

	data := response.Data
	assert.IsType(t, []*FirecrawlDocument{}, data)

	assert.Greater(t, len(data), 0)
	assert.Contains(t, data[0].Markdown, "# Scrape This Site")
	assert.NotNil(t, data[0].Metadata)
}

func TestCrawlURLWithOptionsE2E(t *testing.T) {
	app, err := NewFirecrawlApp(TEST_API_KEY, API_URL)
	require.NoError(t, err)

	response, err := app.CrawlURL("https://www.scrapethissite.com",
		&CrawlParams{
			ExcludePaths:       []string{"blog/*"},
			IncludePaths:       []string{"/"},
			MaxDepth:           ptr(2),
			IgnoreSitemap:      ptr(true),
			Limit:              ptr(10),
			AllowBackwardLinks: ptr(true),
			AllowExternalLinks: ptr(true),
			ScrapeOptions: ScrapeParams{
				Formats:         []string{"markdown", "html", "rawHtml", "screenshot", "links"},
				Headers:         ptr(map[string]string{"x-key": "test"}),
				IncludeTags:     []string{"h1"},
				ExcludeTags:     []string{"h2"},
				OnlyMainContent: ptr(true),
				WaitFor:         ptr(1000),
			},
		},
		nil,
	)
	require.NoError(t, err)
	assert.NotNil(t, response)

	assert.Greater(t, response.Total, 0)
	assert.Greater(t, response.Completed, 0)
	assert.Greater(t, response.CreditsUsed, 0)
	assert.NotEmpty(t, response.ExpiresAt)
	assert.Equal(t, response.Status, "completed")

	data := response.Data
	assert.IsType(t, []*FirecrawlDocument{}, data)

	assert.Greater(t, len(data), 0)
	assert.Contains(t, data[0].Markdown, "# Scrape This Site")
	assert.NotNil(t, data[0].Metadata)
	assert.Contains(t, data[0].HTML, "<h1")
	assert.Contains(t, data[0].RawHTML, "<h1")
	assert.Contains(t, data[0].Screenshot, "https://")
	assert.NotNil(t, data[0].Links)
	assert.NotNil(t, data[0].Metadata.Title)
	assert.NotNil(t, data[0].Metadata.Description)
	assert.NotNil(t, data[0].Metadata.Language)
	assert.NotNil(t, data[0].Metadata.SourceURL)
	assert.NotNil(t, data[0].Metadata.StatusCode)
	assert.Equal(t, 200, *data[0].Metadata.StatusCode)
	assert.Empty(t, data[0].Metadata.Error)
}

func TestCrawlURLWithIdempotencyKeyE2E(t *testing.T) {
	app, err := NewFirecrawlApp(TEST_API_KEY, API_URL)
	require.NoError(t, err)

	uniqueIdempotencyKey := uuid.New().String()
	params := &CrawlParams{
		ExcludePaths: []string{"blog/*"},
		Limit:        ptr(10),
	}
	response, err := app.CrawlURL("https://www.scrapethissite.com", params, &uniqueIdempotencyKey)
	require.NoError(t, err)
	assert.NotNil(t, response)

	data := response.Data
	require.Greater(t, len(data), 0)
	require.IsType(t, []*FirecrawlDocument{}, data)
	assert.Contains(t, data[0].Markdown, "# Scrape This Site")

	_, err = app.CrawlURL("https://firecrawl.dev", params, &uniqueIdempotencyKey)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Conflict: Failed to start crawl job due to a conflict. Idempotency key already used")
}

func TestAsyncCrawlURLE2E(t *testing.T) {
	app, err := NewFirecrawlApp(TEST_API_KEY, API_URL)
	require.NoError(t, err)

	response, err := app.AsyncCrawlURL("https://www.scrapethissite.com", nil, nil)
	require.NoError(t, err)
	assert.NotNil(t, response)

	assert.NotEmpty(t, response.ID)
	assert.NotEmpty(t, response.URL)
	assert.True(t, response.Success)
}

func TestAsyncCrawlURLWithOptionsE2E(t *testing.T) {
	app, err := NewFirecrawlApp(TEST_API_KEY, API_URL)
	require.NoError(t, err)

	response, err := app.AsyncCrawlURL("https://www.scrapethissite.com",
		&CrawlParams{
			ExcludePaths:       []string{"blog/*"},
			IncludePaths:       []string{"/"},
			MaxDepth:           ptr(2),
			IgnoreSitemap:      ptr(true),
			Limit:              ptr(10),
			AllowBackwardLinks: ptr(true),
			AllowExternalLinks: ptr(true),
			ScrapeOptions: ScrapeParams{
				Formats:         []string{"markdown", "html", "rawHtml", "screenshot", "links"},
				Headers:         ptr(map[string]string{"x-key": "test"}),
				IncludeTags:     []string{"h1"},
				ExcludeTags:     []string{"h2"},
				OnlyMainContent: ptr(true),
				WaitFor:         ptr(1000),
			},
		},
		nil,
	)
	require.NoError(t, err)
	assert.NotNil(t, response)

	assert.NotEmpty(t, response.ID)
	assert.NotEmpty(t, response.URL)
	assert.True(t, response.Success)
}

func TestAsyncCrawlURLWithIdempotencyKeyE2E(t *testing.T) {
	app, err := NewFirecrawlApp(TEST_API_KEY, API_URL)
	require.NoError(t, err)

	uniqueIdempotencyKey := uuid.New().String()
	params := &CrawlParams{
		ExcludePaths: []string{"blog/*"},
	}
	response, err := app.AsyncCrawlURL("https://www.scrapethissite.com", params, &uniqueIdempotencyKey)
	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.NotNil(t, response.ID)
	assert.NotNil(t, response.URL)
	assert.True(t, response.Success)

	_, err = app.AsyncCrawlURL("https://firecrawl.dev", params, &uniqueIdempotencyKey)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Conflict: Failed to start crawl job due to a conflict. Idempotency key already used")
}

func TestCheckCrawlStatusE2E(t *testing.T) {
	app, err := NewFirecrawlApp(TEST_API_KEY, API_URL)
	require.NoError(t, err)

	params := &CrawlParams{
		ScrapeOptions: ScrapeParams{
			Formats: []string{"markdown", "html", "rawHtml", "screenshot", "links"},
		},
	}
	asyncCrawlResponse, err := app.AsyncCrawlURL("https://firecrawl.dev", params, nil)
	require.NoError(t, err)
	assert.NotNil(t, asyncCrawlResponse)

	const maxChecks = 15
	checks := 0

	for {
		if checks >= maxChecks {
			break
		}

		time.Sleep(5 * time.Second) // wait for 5 seconds

		response, err := app.CheckCrawlStatus(asyncCrawlResponse.ID)
		require.NoError(t, err)
		assert.NotNil(t, response)

		assert.GreaterOrEqual(t, len(response.Data), 0)
		assert.GreaterOrEqual(t, response.Total, 0)
		assert.GreaterOrEqual(t, response.CreditsUsed, 0)

		if response.Status == "completed" {
			break
		}

		checks++
	}

	// Final check after loop or if completed
	response, err := app.CheckCrawlStatus(asyncCrawlResponse.ID)
	require.NoError(t, err)
	assert.NotNil(t, response)

	assert.Equal(t, "completed", response.Status)
	assert.Greater(t, len(response.Data), 0)
	assert.Greater(t, response.Total, 0)
	assert.Greater(t, response.Completed, 0)
	assert.Greater(t, response.CreditsUsed, 0)
	assert.NotNil(t, response.Data[0].Markdown)
	assert.Contains(t, response.Data[0].HTML, "<div")
	assert.Contains(t, response.Data[0].RawHTML, "<div")
	assert.Contains(t, response.Data[0].Screenshot, "https://")
	assert.NotNil(t, response.Data[0].Links)
	assert.Greater(t, len(response.Data[0].Links), 0)
	assert.NotNil(t, response.Data[0].Metadata.Title)
	assert.NotNil(t, response.Data[0].Metadata.Description)
	assert.NotNil(t, response.Data[0].Metadata.Language)
	assert.NotNil(t, response.Data[0].Metadata.SourceURL)
	assert.NotNil(t, response.Data[0].Metadata.StatusCode)
	assert.Empty(t, response.Data[0].Metadata.Error)
}

func TestMapURLInvalidAPIKey(t *testing.T) {
	invalidApp, err := NewFirecrawlApp("invalid_api_key", API_URL)
	require.NoError(t, err)
	_, err = invalidApp.MapURL("https://www.scrapethissite.com", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Status code 401")
}

func TestMapURLBlocklistedURL(t *testing.T) {
	app, err := NewFirecrawlApp(TEST_API_KEY, API_URL)
	require.NoError(t, err)
	blocklistedUrl := "https://facebook.com/fake-test"
	_, err = app.MapURL(blocklistedUrl, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Status code 403")
}

func TestMapURLValidMap(t *testing.T) {
	app, err := NewFirecrawlApp(TEST_API_KEY, API_URL)
	require.NoError(t, err)

	response, err := app.MapURL("https://www.scrapethissite.com", nil)
	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.IsType(t, &MapResponse{}, response)
	assert.Greater(t, len(response.Links), 0)
	assert.Contains(t, response.Links[0], "https://")
	assert.Contains(t, response.Links[0], "scrapethissite.com")
}

func TestMapURLWithSearchParameter(t *testing.T) {
	app, err := NewFirecrawlApp(TEST_API_KEY, API_URL)
	require.NoError(t, err)

	_, err = app.Search("https://www.scrapethissite.com", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Search is not implemented in API version 1.0.0")
}
