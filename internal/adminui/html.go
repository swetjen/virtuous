package adminui

import (
	"html"
	"strconv"
	"strings"
)

const scalarCDNURL = "https://cdn.jsdelivr.net/npm/@scalar/api-reference@1.57.5"

// DocsShellOptions configures the docs UI shell.
type DocsShellOptions struct {
	Title            string
	OpenAPIURL       string
	Modules          []string
	EventsURL        string
	EventsStreamURL  string
	LoggingStatusURL string
	MetricsURL       string
}

// DocsShellHTML renders the Scalar-powered docs shell HTML.
func DocsShellHTML(opts DocsShellOptions) string {
	title := strings.TrimSpace(opts.Title)
	if title == "" {
		title = "Virtuous Docs"
	}
	openAPIURL := strings.TrimSpace(opts.OpenAPIURL)
	if openAPIURL == "" {
		openAPIURL = "/openapi.json"
	}

	moduleAPI := true
	if len(opts.Modules) > 0 {
		moduleAPI = false
		for _, module := range opts.Modules {
			if strings.EqualFold(strings.TrimSpace(module), "api") {
				moduleAPI = true
				break
			}
		}
	}

	replacer := strings.NewReplacer(
		"__TITLE__", html.EscapeString(title),
		"__OPENAPI_URL__", strconv.Quote(openAPIURL),
		"__SCALAR_CDN_URL__", scalarCDNURL,
		"__MODULE_API__", strconv.FormatBool(moduleAPI),
	)
	return replacer.Replace(docsShellTemplate)
}

const docsShellTemplate = `<!doctype html>
<html lang="en">
<head>
	<meta charset="utf-8" />
	<meta name="viewport" content="width=device-width, initial-scale=1" />
	<title>__TITLE__</title>
	<style>
		body {
			margin: 0;
			font-family: Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
			background: #fff;
		}

		#app {
			min-height: 100vh;
		}

		.empty {
			min-height: 100vh;
			display: grid;
			place-items: center;
			padding: 32px;
			color: #475569;
			background: #f8fafc;
			text-align: center;
		}

		.empty h1 {
			margin: 0 0 8px;
			color: #0f172a;
			font-size: 24px;
		}

		.empty p {
			margin: 0;
			max-width: 560px;
			line-height: 1.5;
		}
	</style>
</head>
<body>
	<div id="app"></div>
	<script src="__SCALAR_CDN_URL__"></script>
	<script>
		const OPENAPI_URL = __OPENAPI_URL__
		const MODULE_API = __MODULE_API__

		if (!MODULE_API) {
			document.getElementById("app").innerHTML =
				"<main class='empty'><div><h1>API docs disabled</h1><p>The API module is disabled for this docs mount.</p></div></main>"
		} else {
			Scalar.createApiReference("#app", {
				url: OPENAPI_URL,
				layout: "modern",
				theme: "default",
				withDefaultFonts: false,
				telemetry: false,
				persistAuth: false,
				agent: {
					disabled: true,
				},
			})
		}
	</script>
</body>
</html>`
