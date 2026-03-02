package adminui

import (
	"html"
	"strconv"
	"strings"
)

// DocsShellOptions configures the integrated docs/admin UI shell.
type DocsShellOptions struct {
	Title            string
	OpenAPIURL       string
	SQLCatalogURL    string
	EventsURL        string
	EventsStreamURL  string
	LoggingStatusURL string
	MetricsURL       string
}

// DocsShellHTML renders the docs/admin shell HTML.
func DocsShellHTML(opts DocsShellOptions) string {
	title := strings.TrimSpace(opts.Title)
	if title == "" {
		title = "Virtuous Docs"
	}
	openAPIURL := strings.TrimSpace(opts.OpenAPIURL)
	if openAPIURL == "" {
		openAPIURL = "/openapi.json"
	}
	sqlCatalogURL := strings.TrimSpace(opts.SQLCatalogURL)
	if sqlCatalogURL == "" {
		sqlCatalogURL = "./_admin/sql"
	}
	eventsURL := strings.TrimSpace(opts.EventsURL)
	if eventsURL == "" {
		eventsURL = "./_admin/events"
	}
	eventsStreamURL := strings.TrimSpace(opts.EventsStreamURL)
	if eventsStreamURL == "" {
		eventsStreamURL = "./_admin/events.stream"
	}
	loggingStatusURL := strings.TrimSpace(opts.LoggingStatusURL)
	if loggingStatusURL == "" {
		loggingStatusURL = "./_admin/logging"
	}
	metricsURL := strings.TrimSpace(opts.MetricsURL)
	if metricsURL == "" {
		metricsURL = "/rpc/_virtuous/metrics"
	}

	replacer := strings.NewReplacer(
		"__TITLE__", html.EscapeString(title),
		"__OPENAPI_URL__", strconv.Quote(openAPIURL),
		"__SQL_CATALOG_URL__", strconv.Quote(sqlCatalogURL),
		"__EVENTS_URL__", strconv.Quote(eventsURL),
		"__EVENTS_STREAM_URL__", strconv.Quote(eventsStreamURL),
		"__LOGGING_STATUS_URL__", strconv.Quote(loggingStatusURL),
		"__METRICS_URL__", strconv.Quote(metricsURL),
	)
	return replacer.Replace(docsShellTemplate)
}

const docsShellTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8" />
	<meta name="viewport" content="width=device-width, initial-scale=1" />
	<title>__TITLE__</title>
	<style>
		* {
			box-sizing: border-box;
		}

		:root {
			--bg: #edf3f8;
			--panel: #ffffff;
			--text: #11263a;
			--muted: #536579;
			--line: #c4d4e2;
			--brand: #0d7a72;
			--brand-strong: #0a5f59;
			--ok: #0f9d58;
			--warn: #c07d13;
			--err: #bd2d2d;
			--chip: #e6eef5;
		}

		body {
			margin: 0;
			min-height: 100vh;
			font-family: "Space Grotesk", "Avenir Next", "Segoe UI", sans-serif;
			color: var(--text);
			background:
				radial-gradient(circle at 0 0, #dff2ff 0, rgba(223, 242, 255, 0) 40%),
				radial-gradient(circle at 100% 0, #defbe8 0, rgba(222, 251, 232, 0) 35%),
				var(--bg);
		}

		.topbar {
			--topbar-height: 58px;
			height: var(--topbar-height);
			display: flex;
			align-items: center;
			justify-content: space-between;
			gap: 14px;
			padding: 0 14px;
			background: rgba(244, 248, 252, 0.92);
			border-bottom: 1px solid #d3e0ec;
			backdrop-filter: blur(8px);
			position: sticky;
			top: 0;
			z-index: 20;
		}

		.brand-block {
			display: flex;
			align-items: baseline;
			gap: 8px;
			min-width: 0;
		}

		.brand {
			font-size: 15px;
			font-weight: 700;
			letter-spacing: 0.01em;
			white-space: nowrap;
		}

		.brand-sub {
			font-size: 12px;
			color: #597088;
			white-space: nowrap;
		}

		.nav {
			display: flex;
			align-items: center;
			gap: 6px;
		}

		.nav button {
			all: unset;
			cursor: pointer;
			padding: 8px 12px;
			border-radius: 999px;
			color: #38526a;
			font-size: 13px;
			font-weight: 600;
			line-height: 1;
			background: transparent;
			border: 1px solid transparent;
		}

		.nav button:hover {
			background: #e8f0f7;
			border-color: #d4e1ed;
		}

		.nav button.active {
			background: #112b42;
			color: #edf5ff;
			border-color: #112b42;
		}

		.main {
			padding: 16px;
			min-height: calc(100vh - 58px);
		}

		.panel-shell {
			background: var(--panel);
			border: 1px solid var(--line);
			border-radius: 14px;
			box-shadow: 0 20px 56px rgba(16, 41, 67, 0.12);
			overflow: hidden;
			min-height: calc(100vh - 90px);
			display: flex;
			flex-direction: column;
		}

		body.reference-mode .main {
			padding: 0;
		}

		body.reference-mode .panel-shell {
			border: 0;
			border-radius: 0;
			box-shadow: none;
			min-height: calc(100vh - 58px);
		}

		body.reference-mode .tiles {
			display: none;
		}

		body.reference-mode #panel-reference {
			padding: 0;
			gap: 0;
		}

		body.reference-mode #panel-reference .section-head {
			display: none;
		}

		body.reference-mode #panel-reference .card {
			border: 0;
			border-radius: 0;
			flex: 1;
		}

		body.reference-mode #scalar-root {
			min-height: calc(100vh - 58px);
			height: 100%;
		}

		body.logs-mode .main {
			padding: 0;
		}

		body.logs-mode .panel-shell {
			border: 0;
			border-radius: 0;
			box-shadow: none;
			background: #0a1118;
			min-height: calc(100vh - 58px);
		}

		body.logs-mode .tiles {
			background: #101a24;
			border-bottom: 1px solid #1b2e40;
		}

		body.logs-mode .tile {
			border: 0;
			background: #162533;
		}

		body.logs-mode .tile h3 {
			color: #9cb2c6;
		}

		body.logs-mode .tile p {
			color: #e4edf7;
		}

		body.logs-mode .stream-state {
			background: #1a2c3c;
			color: #a8bdd0;
		}

		.tiles {
			display: grid;
			grid-template-columns: repeat(4, minmax(0, 1fr));
			gap: 10px;
			padding: 12px;
			background: linear-gradient(180deg, #f8fbfe 0%, #f3f8fc 100%);
			border-bottom: 1px solid var(--line);
		}

		.tile {
			padding: 12px;
			border-radius: 12px;
			background: #ffffff;
			border: 1px solid #d6e3ef;
		}

		.tile h3 {
			margin: 0;
			font-size: 12px;
			font-weight: 700;
			text-transform: uppercase;
			letter-spacing: 0.08em;
			color: var(--muted);
		}

		.tile p {
			margin: 8px 0 0;
			font-size: 22px;
			font-weight: 700;
		}

		.tile p.ok {
			color: var(--ok);
		}

		.tile p.warn {
			color: var(--warn);
		}

		.tile p.err {
			color: var(--err);
		}

		.spark-wrap {
			display: flex;
			align-items: center;
			gap: 10px;
			margin-top: 8px;
		}

		.sparkline {
			display: grid;
			grid-template-columns: repeat(48, minmax(2px, 1fr));
			gap: 2px;
			width: 100%;
			height: 18px;
		}

		.spark-bar {
			border-radius: 2px;
			opacity: 0.85;
			background: #cad8e5;
		}

		.spark-bar.ok {
			background: #2abf74;
		}

		.spark-bar.invalid {
			background: #f2aa35;
		}

		.spark-bar.err {
			background: #df4545;
		}

		.stream-state {
			font-size: 12px;
			padding: 4px 8px;
			border-radius: 999px;
			background: var(--chip);
			color: var(--muted);
			white-space: nowrap;
		}

		.panel {
			display: none;
			padding: 12px;
			gap: 12px;
			flex: 1;
			overflow: auto;
		}

		.panel.active {
			display: flex;
			flex-direction: column;
		}

		.section-head {
			display: flex;
			justify-content: space-between;
			align-items: center;
			gap: 12px;
		}

		.section-head h2 {
			margin: 0;
			font-size: 18px;
		}

		.section-head p {
			margin: 4px 0 0;
			font-size: 13px;
			color: var(--muted);
		}

		.card {
			border: 1px solid var(--line);
			border-radius: 12px;
			background: #ffffff;
			overflow: hidden;
		}

		.card-head {
			display: flex;
			justify-content: space-between;
			align-items: center;
			gap: 10px;
			padding: 12px 14px;
			border-bottom: 1px solid #e5edf4;
			background: #f8fbfe;
		}

		.card-head h3 {
			margin: 0;
			font-size: 14px;
		}

		.card-head p {
			margin: 4px 0 0;
			font-size: 12px;
			color: var(--muted);
		}

		.metric-pill {
			display: inline-flex;
			align-items: center;
			gap: 6px;
			padding: 4px 8px;
			border-radius: 999px;
			background: #e6eef5;
			color: #30495f;
			font-size: 11px;
			font-weight: 700;
			text-transform: uppercase;
			letter-spacing: 0.06em;
			white-space: nowrap;
		}

		.metric-pill.advanced {
			background: #dff5ea;
			color: #0b7b4a;
		}

		.metric-pill.basic {
			background: #edf1f5;
			color: #52677a;
		}

		#scalar-root {
			min-height: 680px;
		}

		.observability-grid {
			display: grid;
			grid-template-columns: repeat(2, minmax(0, 1fr));
			gap: 12px;
		}

		.observability-card-body {
			padding: 0;
		}

		.observability-card-body table {
			font-size: 12px;
		}

		.observability-card-body tbody td {
			white-space: nowrap;
		}

		.observability-card-body tbody td.rpc {
			max-width: 260px;
			overflow: hidden;
			text-overflow: ellipsis;
			font-weight: 600;
			color: #17324a;
		}

		.observability-card-body tbody td.message {
			max-width: 420px;
			overflow: hidden;
			text-overflow: ellipsis;
		}

		.observability-empty {
			padding: 18px 16px;
			font-size: 13px;
			color: var(--muted);
		}

		.observability-note {
			margin: 0;
			padding: 12px 14px;
			font-size: 12px;
			color: var(--muted);
			border-top: 1px solid #e5edf4;
			background: #fbfdff;
		}

		.sql-layout {
			display: grid;
			grid-template-columns: 240px 240px minmax(0, 1fr);
			gap: 10px;
			min-height: 560px;
		}

		.sql-list {
			display: flex;
			flex-direction: column;
			border: 1px solid var(--line);
			border-radius: 12px;
			overflow: hidden;
			background: #f8fbfd;
		}

		.sql-list h3 {
			margin: 0;
			padding: 10px 12px;
			font-size: 13px;
			text-transform: uppercase;
			letter-spacing: 0.06em;
			background: #eef4f9;
			border-bottom: 1px solid var(--line);
		}

		.sql-items {
			overflow: auto;
		}

		.sql-item {
			display: block;
			width: 100%;
			text-align: left;
			background: transparent;
			border: 0;
			border-bottom: 1px solid #dde7f0;
			padding: 10px 12px;
			font-size: 12px;
			cursor: pointer;
			color: #23384e;
		}

		.sql-item small {
			display: block;
			color: #627589;
			margin-top: 2px;
		}

		.sql-item:hover,
		.sql-item.active {
			background: #dff0fb;
		}

		.sql-viewer {
			display: flex;
			flex-direction: column;
			border: 1px solid var(--line);
			border-radius: 12px;
			overflow: hidden;
			background: #f6fafc;
		}

		.sql-meta {
			padding: 10px 12px;
			background: #eef4f9;
			border-bottom: 1px solid var(--line);
			display: flex;
			justify-content: space-between;
			align-items: center;
			gap: 8px;
			font-size: 12px;
		}

		#sql-content {
			margin: 0;
			padding: 12px;
			font-family: "IBM Plex Mono", "SFMono-Regular", Menlo, monospace;
			font-size: 12px;
			line-height: 1.5;
			overflow: auto;
			white-space: pre;
			background: #f6fafc;
			color: #1c3349;
			min-height: 520px;
		}

		table {
			width: 100%;
			border-collapse: collapse;
			font-size: 12px;
		}

		thead th {
			position: sticky;
			top: 0;
			background: #eef4f9;
			text-align: left;
			padding: 10px;
			border-bottom: 1px solid var(--line);
			color: #243a50;
			z-index: 1;
		}

		tbody td {
			padding: 8px 10px;
			border-bottom: 1px solid #ebf1f6;
			vertical-align: top;
		}

		tbody tr.outcome-ok td.status {
			color: var(--ok);
			font-weight: 700;
		}

		tbody tr.outcome-invalid td.status {
			color: var(--warn);
			font-weight: 700;
		}

		tbody tr.outcome-err td.status {
			color: var(--err);
			font-weight: 700;
		}

		tbody tr.system-row {
			background: #f8fbfd;
		}

		#panel-logs {
			padding: 0;
			gap: 0;
			background: #0b131b;
			color: #d9e3ed;
			border-radius: 12px;
			overflow: hidden;
		}

		#panel-logs .section-head {
			padding: 14px 16px;
			background: #101b26;
			border-bottom: 1px solid #1f3346;
		}

		#panel-logs .section-head h2 {
			color: #ebf4ff;
		}

		#panel-logs .section-head p {
			color: #91a8bd;
		}

		#panel-logs .logs-table-wrap {
			border: 0;
			border-radius: 0;
			overflow: auto;
			background: #0d1620;
			flex: 1;
		}

		#panel-logs .logs-disabled {
			margin: 20px;
			padding: 16px;
			border-radius: 12px;
			background: #111f2c;
			border: 1px solid #22384c;
		}

		#panel-logs .logs-disabled h3 {
			margin: 0;
			font-size: 16px;
			color: #e8f2fc;
		}

		#panel-logs .logs-disabled p {
			margin: 8px 0 0;
			font-size: 13px;
			color: #9eb4c8;
		}

		#panel-logs .logs-disabled pre {
			margin: 12px 0 0;
			padding: 12px;
			border-radius: 10px;
			background: #0c161f;
			border: 1px solid #1d3042;
			color: #b7cde2;
			font-size: 12px;
			line-height: 1.5;
			overflow: auto;
			white-space: pre;
		}

		#panel-logs table {
			font-family: "IBM Plex Mono", "SFMono-Regular", Menlo, monospace;
			color: #d2ddea;
		}

		#panel-logs thead th {
			background: #121f2b;
			color: #9ab0c4;
			border-bottom: 1px solid #22384c;
		}

		#panel-logs tbody td {
			border-bottom: 1px solid #192b3a;
			color: #d0dbe8;
		}

		#panel-logs tbody tr:hover td {
			background: #142433;
		}

		#panel-logs tbody tr.outcome-ok td.status {
			color: #43d182;
			font-weight: 700;
		}

		#panel-logs tbody tr.outcome-invalid td.status {
			color: #f2b640;
			font-weight: 700;
		}

		#panel-logs tbody tr.outcome-err td.status {
			color: #ff6f6f;
			font-weight: 700;
		}

		#panel-logs tbody tr.system-row td {
			background: #101a24;
			color: #8da3b7;
		}

		#panel-logs .empty {
			color: #8da3b7;
		}

		.empty {
			padding: 16px;
			font-size: 13px;
			color: var(--muted);
		}

		@media (max-width: 1100px) {
			.topbar {
				padding: 0 10px;
			}

			.brand-sub {
				display: none;
			}

			.main {
				padding: 0;
			}

			.panel-shell {
				border-radius: 0;
				min-height: calc(100vh - 58px);
			}
		}

		@media (max-width: 900px) {
			.topbar {
				height: auto;
				min-height: 58px;
				flex-wrap: wrap;
				align-items: center;
				padding: 8px 10px;
			}

			.nav {
				width: 100%;
				overflow-x: auto;
				padding-bottom: 2px;
			}

			.nav button {
				flex: 0 0 auto;
			}

			.tiles {
				grid-template-columns: repeat(2, minmax(0, 1fr));
			}

			.sql-layout {
				grid-template-columns: 1fr;
				min-height: 0;
			}

			.observability-grid {
				grid-template-columns: 1fr;
			}

			#scalar-root {
				min-height: 420px;
			}

			#sql-content {
				min-height: 360px;
			}
		}
	</style>
</head>
<body>
	<header class="topbar">
		<div class="brand-block">
			<div class="brand">Virtuous Console</div>
			<div class="brand-sub">docs + observability + sql + runtime logs</div>
		</div>
		<nav class="nav" aria-label="Docs sections">
			<button class="active" data-panel="reference">API Reference</button>
			<button data-panel="observability">Observability</button>
			<button data-panel="database">Database</button>
			<button data-panel="logs">Live Logs</button>
		</nav>
	</header>

	<main class="main">
			<div class="panel-shell">
				<section class="tiles" aria-label="Request summary">
					<article class="tile">
						<h3>Total</h3>
						<p id="tile-total">0</p>
					</article>
					<article class="tile">
						<h3>OK</h3>
						<p id="tile-ok" class="ok">0</p>
					</article>
					<article class="tile">
						<h3>Invalid</h3>
						<p id="tile-invalid" class="warn">0</p>
					</article>
					<article class="tile">
						<h3>Server Err</h3>
						<p id="tile-err" class="err">0</p>
					</article>
					<div class="spark-wrap" style="grid-column: 1 / -1;">
						<div id="sparkline" class="sparkline" aria-label="Recent request outcomes"></div>
						<span id="stream-state" class="stream-state">offline</span>
					</div>
				</section>

				<section id="panel-reference" class="panel active">
					<div class="section-head">
						<div>
							<h2>API Reference</h2>
							<p>Scalar-rendered OpenAPI explorer.</p>
						</div>
					</div>
					<div class="card">
						<div id="scalar-root"></div>
					</div>
				</section>

				<section id="panel-observability" class="panel">
					<div class="section-head">
						<div>
							<h2>Observability</h2>
							<p>RPC-native request, error, guard, and latency summaries.</p>
						</div>
						<div style="display:flex; gap:8px; align-items:center; flex-wrap:wrap;">
							<span id="obs-mode" class="metric-pill basic">basic</span>
							<span id="obs-updated" class="stream-state">waiting</span>
						</div>
					</div>

					<div class="observability-grid">
						<div class="card">
							<div class="card-head">
								<div>
									<h3>Error Summary</h3>
									<p>Server/client error pressure by RPC over the last 24 hours.</p>
								</div>
							</div>
							<div class="observability-card-body">
								<table>
									<thead>
										<tr>
											<th>RPC</th>
											<th>Server Err</th>
											<th>Client Err</th>
											<th>Error Rate</th>
										</tr>
									</thead>
									<tbody id="obs-error-summary-rows">
										<tr><td colspan="4" class="observability-empty">Loading metrics...</td></tr>
									</tbody>
								</table>
							</div>
						</div>

						<div class="card">
							<div class="card-head">
								<div>
									<h3>Current Load</h3>
									<p>Per-RPC traffic and latency over 1 minute, 1 hour, and 24 hours.</p>
								</div>
							</div>
							<div class="observability-card-body">
								<table>
									<thead>
										<tr>
											<th>RPC</th>
											<th>Req/min</th>
											<th>Req/24h</th>
											<th>Avg ms</th>
											<th>P95 ms</th>
											<th>Traces</th>
										</tr>
									</thead>
									<tbody id="obs-load-rows">
										<tr><td colspan="6" class="observability-empty">Loading metrics...</td></tr>
									</tbody>
								</table>
							</div>
						</div>
					</div>

					<div class="observability-grid">
						<div class="card">
							<div class="card-head">
								<div>
									<h3>Application Errors</h3>
									<p>Repeated 5xx failures grouped by message and stack signature.</p>
								</div>
								<span id="obs-sample-rate" class="stream-state">sampling --</span>
							</div>
							<div class="observability-card-body">
								<table>
									<thead>
										<tr>
											<th>RPC</th>
											<th>Error</th>
											<th>Count</th>
											<th>Sparkline</th>
										</tr>
									</thead>
									<tbody id="obs-errors-rows">
										<tr><td colspan="4" class="observability-empty">Advanced observability is disabled.</td></tr>
									</tbody>
								</table>
							</div>
						</div>

						<div class="card">
							<div class="card-head">
								<div>
									<h3>Guard Metrics</h3>
									<p>Allow/deny outcomes for evaluated guards across the last 24 hours.</p>
								</div>
							</div>
							<div class="observability-card-body">
								<table>
									<thead>
										<tr>
											<th>RPC</th>
											<th>Guard</th>
											<th>Denied</th>
											<th>Denial %</th>
										</tr>
									</thead>
									<tbody id="obs-guards-rows">
										<tr><td colspan="4" class="observability-empty">Advanced observability is disabled.</td></tr>
									</tbody>
								</table>
							</div>
						</div>
					</div>

					<div class="card">
						<div class="card-head">
							<div>
								<h3>Recent Traces</h3>
								<p>Sampled request snapshots. Full timeline drilldown is deferred in this MVP.</p>
							</div>
						</div>
						<div class="observability-card-body">
							<table>
								<thead>
									<tr>
										<th>Time</th>
										<th>RPC</th>
										<th>Status</th>
										<th>Duration</th>
										<th>Guard</th>
										<th>Error</th>
									</tr>
								</thead>
								<tbody id="obs-traces-rows">
									<tr><td colspan="6" class="observability-empty">Advanced observability is disabled.</td></tr>
								</tbody>
							</table>
						</div>
						<p class="observability-note">Trace capture stores a bounded in-memory sample and is cleared on restart.</p>
					</div>
				</section>

				<section id="panel-database" class="panel">
					<div class="section-head">
						<div>
							<h2>Database Explorer</h2>
							<p>Goose migrations + SQLC queries discovered from db/sql.</p>
						</div>
						<span id="sql-root" class="stream-state"></span>
					</div>
					<div id="sql-status" class="empty">Loading SQL catalog...</div>
					<div id="sql-layout" class="sql-layout" hidden>
						<div class="sql-list">
							<h3>Schemas</h3>
							<div id="schemas-list" class="sql-items"></div>
						</div>
						<div class="sql-list">
							<h3>Queries</h3>
							<div id="queries-list" class="sql-items"></div>
						</div>
						<div class="sql-viewer">
							<div class="sql-meta">
								<strong id="sql-file-path">Select a SQL file</strong>
								<span id="sql-file-details"></span>
							</div>
							<pre id="sql-content"></pre>
						</div>
					</div>
				</section>

				<section id="panel-logs" class="panel">
					<div class="section-head">
						<div>
							<h2>Live Logs</h2>
							<p>Recent events with route method, response code, and latency.</p>
						</div>
					</div>
					<div id="logs-disabled" class="logs-disabled" hidden>
						<h3>Virtuous logger is not attached.</h3>
						<p>Attach it once at your top-level mux/handler boundary to enable live logs.</p>
						<pre id="logs-snippet"></pre>
					</div>
					<div id="logs-table-wrap" class="logs-table-wrap">
						<table>
							<thead>
								<tr>
									<th>Time</th>
									<th>Method</th>
									<th>Path / Message</th>
									<th>Status</th>
									<th>Duration</th>
									<th>Bytes</th>
									<th>Outcome</th>
								</tr>
							</thead>
							<tbody id="log-rows">
								<tr><td colspan="7" class="empty">Waiting for events...</td></tr>
							</tbody>
						</table>
					</div>
				</section>
			</div>
	</main>

	<script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script>
	<script>
	(function () {
		const OPENAPI_URL = __OPENAPI_URL__
		const SQL_CATALOG_URL = __SQL_CATALOG_URL__
		const EVENTS_URL = __EVENTS_URL__
		const EVENTS_STREAM_URL = __EVENTS_STREAM_URL__
		const LOGGING_STATUS_URL = __LOGGING_STATUS_URL__
		const METRICS_URL = __METRICS_URL__
		const MAX_EVENTS = 600
		const events = []
		let latestEventID = 0
		let scalarMounted = false
		let streamConnected = false
		let eventSource = null
		let reconnectTimer = null
		let selected = { kind: "", index: 0 }
		let sqlCatalog = { schemas: [], queries: [] }
		let loggingStatus = { enabled: false, active: false, snippet: "" }
		let metricsSnapshot = null

		const navButtons = Array.from(document.querySelectorAll(".nav button"))
		const panels = {
			reference: document.getElementById("panel-reference"),
			observability: document.getElementById("panel-observability"),
			database: document.getElementById("panel-database"),
			logs: document.getElementById("panel-logs"),
		}

		function setStreamState(state) {
			const node = document.getElementById("stream-state")
			node.textContent = state
			if (state === "live") {
				node.style.background = "#d9f7e6"
				node.style.color = "#0f7d46"
				return
			}
			if (state === "disabled") {
				node.style.background = "#efe7da"
				node.style.color = "#7b5b2f"
				return
			}
			node.style.background = "#e6eef5"
			node.style.color = "#556578"
		}

		function showPanel(name) {
			navButtons.forEach(function (button) {
				button.classList.toggle("active", button.getAttribute("data-panel") === name)
			})
			Object.keys(panels).forEach(function (key) {
				panels[key].classList.toggle("active", key === name)
			})
			document.body.classList.toggle("reference-mode", name === "reference")
			document.body.classList.toggle("logs-mode", name === "logs")
			if (name === "reference") {
				mountScalar()
			}
			window.location.hash = name
		}

		navButtons.forEach(function (button) {
			button.addEventListener("click", function () {
				showPanel(button.getAttribute("data-panel"))
			})
		})

		function activePanelFromHash() {
			const hash = (window.location.hash || "").replace("#", "")
			if (hash === "database" || hash === "logs" || hash === "reference" || hash === "observability") {
				return hash
			}
			return "reference"
		}

		function buildPrefixMap(spec) {
			const map = {}
			if (!spec || !spec.components || !spec.components.securitySchemes) {
				return map
			}
			const schemes = spec.components.securitySchemes
			Object.keys(schemes).forEach(function (key) {
				const scheme = schemes[key]
				if (!scheme || scheme.in !== "header" || !scheme.name || !scheme["x-virtuousauth-prefix"]) {
					return
				}
				map[String(scheme.name).toLowerCase()] = String(scheme["x-virtuousauth-prefix"])
			})
			return map
		}

		function installFetchAuthPrefixer(prefixMap) {
			window.__virtuousDocsPrefixMap = prefixMap || {}
			if (window.__virtuousDocsFetchWrapped) {
				return
			}
			window.__virtuousDocsFetchWrapped = true
			const originalFetch = window.fetch.bind(window)
			window.fetch = function (input, init) {
				const activeMap = window.__virtuousDocsPrefixMap || {}
				const headerNames = Object.keys(activeMap)
				if (headerNames.length === 0) {
					return originalFetch(input, init)
				}
				try {
					const sourceHeaders = init && init.headers ? init.headers : (input instanceof Request ? input.headers : undefined)
					const headers = new Headers(sourceHeaders || {})
					let changed = false
					headerNames.forEach(function (headerName) {
						const prefix = activeMap[headerName]
						const current = headers.get(headerName)
						if (!prefix || !current) {
							return
						}
						const expected = prefix + " "
						if (!String(current).startsWith(expected)) {
							headers.set(headerName, expected + current)
							changed = true
						}
					})
					if (!changed) {
						return originalFetch(input, init)
					}
					if (input instanceof Request) {
						const nextInit = init ? Object.assign({}, init) : {}
						nextInit.headers = headers
						return originalFetch(new Request(input, nextInit))
					}
					const nextInit = init ? Object.assign({}, init) : {}
					nextInit.headers = headers
					return originalFetch(input, nextInit)
				} catch (_) {
					return originalFetch(input, init)
				}
			}
		}

		function mountScalar() {
			if (scalarMounted) {
				return
			}
			scalarMounted = true
			fetch(OPENAPI_URL)
				.then(function (response) { return response.json() })
				.then(function (spec) {
					installFetchAuthPrefixer(buildPrefixMap(spec))
					if (typeof Scalar === "undefined" || !Scalar.createApiReference) {
						document.getElementById("scalar-root").innerHTML = "<div class='empty'>Unable to load Scalar API Reference.</div>"
						return
					}
					Scalar.createApiReference("#scalar-root", { url: OPENAPI_URL })
				})
				.catch(function () {
					document.getElementById("scalar-root").innerHTML = "<div class='empty'>Unable to load OpenAPI document.</div>"
				})
		}

		function applyEvents(items) {
			if (!Array.isArray(items)) {
				return
			}
			let changed = false
			items.forEach(function (item) {
				if (!item || typeof item.id !== "number") {
					return
				}
				if (item.id <= latestEventID) {
					return
				}
				latestEventID = item.id
				events.push(item)
				changed = true
			})
			if (!changed) {
				return
			}
			if (events.length > MAX_EVENTS) {
				events.splice(0, events.length - MAX_EVENTS)
			}
			renderMetrics()
			renderLogs()
		}

		function renderMetrics() {
			const requestEvents = events.filter(function (event) { return event.kind === "request" })
			let ok = 0
			let invalid = 0
			let err = 0
			requestEvents.forEach(function (event) {
				if (event.outcome === "err") {
					err += 1
					return
				}
				if (event.outcome === "invalid") {
					invalid += 1
					return
				}
				ok += 1
			})
			document.getElementById("tile-total").textContent = String(requestEvents.length)
			document.getElementById("tile-ok").textContent = String(ok)
			document.getElementById("tile-invalid").textContent = String(invalid)
			document.getElementById("tile-err").textContent = String(err)

			const sparkline = document.getElementById("sparkline")
			sparkline.innerHTML = ""
			requestEvents.slice(-48).forEach(function (event) {
				const bar = document.createElement("span")
				bar.className = "spark-bar " + (event.outcome || "ok")
				bar.title = [event.method || "", event.path || "", event.status || ""].join(" ").trim()
				sparkline.appendChild(bar)
			})
		}

		function formatTime(raw) {
			if (!raw) {
				return "-"
			}
			const date = new Date(raw)
			if (Number.isNaN(date.getTime())) {
				return raw
			}
			return date.toLocaleTimeString()
		}

		function escapeHTML(value) {
			return String(value || "")
				.replace(/&/g, "&amp;")
				.replace(/</g, "&lt;")
				.replace(/>/g, "&gt;")
				.replace(/"/g, "&quot;")
				.replace(/'/g, "&#39;")
		}

		function formatBytes(raw) {
			if (!raw || raw < 1024) {
				return String(raw || 0)
			}
			if (raw < 1024 * 1024) {
				return (raw / 1024).toFixed(1) + " KiB"
			}
			return (raw / (1024 * 1024)).toFixed(1) + " MiB"
		}

		function renderLogs() {
			const tbody = document.getElementById("log-rows")
			if (events.length === 0) {
				tbody.innerHTML = "<tr><td colspan='7' class='empty'>Waiting for events...</td></tr>"
				return
			}
			tbody.innerHTML = ""
			events.slice(-250).reverse().forEach(function (event) {
				const tr = document.createElement("tr")
				if (event.kind === "request") {
					tr.className = "outcome-" + (event.outcome || "ok")
				} else {
					tr.className = "system-row"
				}

				const tdTime = document.createElement("td")
				tdTime.textContent = formatTime(event.time)
				tr.appendChild(tdTime)

				const tdMethod = document.createElement("td")
				tdMethod.textContent = event.kind === "request" ? (event.method || "-") : "system"
				tr.appendChild(tdMethod)

				const tdPath = document.createElement("td")
				tdPath.textContent = event.kind === "request" ? (event.path || "-") : (event.message || "-")
				tr.appendChild(tdPath)

				const tdStatus = document.createElement("td")
				tdStatus.className = "status"
				tdStatus.textContent = event.kind === "request" && event.status ? String(event.status) : "-"
				tr.appendChild(tdStatus)

				const tdDuration = document.createElement("td")
				tdDuration.textContent = event.kind === "request" ? String(event.durationMs || 0) + " ms" : "-"
				tr.appendChild(tdDuration)

				const tdBytes = document.createElement("td")
				tdBytes.textContent = event.kind === "request" ? formatBytes(event.bytes || 0) : "-"
				tr.appendChild(tdBytes)

				const tdOutcome = document.createElement("td")
				tdOutcome.textContent = event.kind === "request" ? (event.outcome || "ok") : "info"
				tr.appendChild(tdOutcome)

				tbody.appendChild(tr)
			})
		}

		function formatMetricTime(raw) {
			if (!raw) {
				return "waiting"
			}
			const date = new Date(raw)
			if (Number.isNaN(date.getTime())) {
				return raw
			}
			return "updated " + date.toLocaleTimeString()
		}

		function formatLatency(raw) {
			const value = Number(raw || 0)
			if (!Number.isFinite(value) || value <= 0) {
				return "0"
			}
			return value.toFixed(value >= 100 ? 0 : 1)
		}

		function formatPercent(raw) {
			const value = Number(raw || 0)
			if (!Number.isFinite(value) || value <= 0) {
				return "0.0%"
			}
			return value.toFixed(1) + "%"
		}

		function routeErrorRate(route) {
			const requests = Number(route && route.requestsLast24h || 0)
			if (requests <= 0) {
				return 0
			}
			const totalErrors = Number(route.serverErrorsLast24h || 0) + Number(route.clientErrorsLast24h || 0)
			return (totalErrors / requests) * 100
		}

		function sparklineHTML(points) {
			if (!Array.isArray(points) || points.length === 0) {
				return "<span class='empty'>-</span>"
			}
			const max = points.reduce(function (current, point) {
				return Math.max(current, Number(point || 0))
			}, 0)
			return "<div class='sparkline'>" + points.map(function (point) {
				const value = Number(point || 0)
				const height = max <= 0 ? 3 : Math.max(3, Math.round((value / max) * 18))
				return "<span class='spark-bar err' style='height:" + String(height) + "px'></span>"
			}).join("") + "</div>"
		}

		function renderTableRows(targetID, emptyColspan, items, buildRow, emptyText) {
			const tbody = document.getElementById(targetID)
			if (!Array.isArray(items) || items.length === 0) {
				tbody.innerHTML = "<tr><td colspan='" + String(emptyColspan) + "' class='observability-empty'>" + emptyText + "</td></tr>"
				return
			}
			tbody.innerHTML = items.map(buildRow).join("")
		}

		function renderObservability() {
			const snapshot = metricsSnapshot
			const mode = document.getElementById("obs-mode")
			const updated = document.getElementById("obs-updated")
			const sampleRate = document.getElementById("obs-sample-rate")
			if (!snapshot) {
				mode.textContent = "offline"
				mode.className = "metric-pill basic"
				updated.textContent = "offline"
				sampleRate.textContent = "sampling --"
				renderTableRows("obs-error-summary-rows", 4, [], function () { return "" }, "Metrics endpoint unavailable.")
				renderTableRows("obs-load-rows", 6, [], function () { return "" }, "Metrics endpoint unavailable.")
				renderTableRows("obs-errors-rows", 4, [], function () { return "" }, "Metrics endpoint unavailable.")
				renderTableRows("obs-guards-rows", 4, [], function () { return "" }, "Metrics endpoint unavailable.")
				renderTableRows("obs-traces-rows", 6, [], function () { return "" }, "Metrics endpoint unavailable.")
				return
			}

			mode.textContent = snapshot.advanced ? "advanced" : "basic"
			mode.className = "metric-pill " + (snapshot.advanced ? "advanced" : "basic")
			updated.textContent = formatMetricTime(snapshot.generatedAt)
			sampleRate.textContent = snapshot.advanced ? ("sampling " + Number((snapshot.sampleRate || 0) * 100).toFixed(0) + "%") : "sampling off"

			const routes = Array.isArray(snapshot.routes) ? snapshot.routes : []
			renderTableRows("obs-error-summary-rows", 4, routes.slice(0, 12), function (route) {
				return "<tr>" +
					"<td class='rpc'>" + escapeHTML(route.rpcName || "-") + "</td>" +
					"<td>" + String(route.serverErrorsLast24h || 0) + "</td>" +
					"<td>" + String(route.clientErrorsLast24h || 0) + "</td>" +
					"<td>" + formatPercent(routeErrorRate(route)) + "</td>" +
				"</tr>"
			}, "No request data captured yet.")

			renderTableRows("obs-load-rows", 6, routes.slice(0, 12), function (route) {
				return "<tr>" +
					"<td class='rpc'>" + escapeHTML(route.rpcName || "-") + "</td>" +
					"<td>" + String(route.requestsLastMinute || 0) + "</td>" +
					"<td>" + String(route.requestsLast24h || 0) + "</td>" +
					"<td>" + formatLatency(route.avgLatencyLastHourMs) + "</td>" +
					"<td>" + formatLatency(route.p95LatencyLastHourMs) + "</td>" +
					"<td>" + String(route.traceSamplesLast24h || 0) + "</td>" +
				"</tr>"
			}, "No request data captured yet.")

			if (!snapshot.advanced) {
				renderTableRows("obs-errors-rows", 4, [], function () { return "" }, "Enable WithAdvancedObservability() to group repeated application errors.")
				renderTableRows("obs-guards-rows", 4, [], function () { return "" }, "Enable WithAdvancedObservability() to capture per-guard allow and deny outcomes.")
				renderTableRows("obs-traces-rows", 6, [], function () { return "" }, "Enable WithAdvancedObservability() to capture sampled traces.")
				return
			}

			renderTableRows("obs-errors-rows", 4, snapshot.errors || [], function (item) {
				const message = item.errorMessage || item.stackSignature || "-"
				return "<tr>" +
					"<td class='rpc'>" + escapeHTML(item.rpcName || "-") + "</td>" +
					"<td class='message' title='" + escapeHTML(message) + "'>" + escapeHTML(message) + "</td>" +
					"<td>" + String(item.countLast24h || 0) + "</td>" +
					"<td>" + sparklineHTML(item.sparkline) + "</td>" +
				"</tr>"
			}, "No application errors captured in the last 24 hours.")

			renderTableRows("obs-guards-rows", 4, snapshot.guards || [], function (item) {
				return "<tr>" +
					"<td class='rpc'>" + escapeHTML(item.rpcName || "-") + "</td>" +
					"<td>" + escapeHTML(item.guardName || "-") + "</td>" +
					"<td>" + String(item.deniedCount || 0) + "</td>" +
					"<td>" + formatPercent(item.denialRatePercent) + "</td>" +
				"</tr>"
			}, "No guard decisions captured in the last 24 hours.")

			renderTableRows("obs-traces-rows", 6, (snapshot.recentTraces || []).slice(0, 16), function (item) {
				return "<tr>" +
					"<td>" + escapeHTML(formatTime(item.timestamp)) + "</td>" +
					"<td class='rpc'>" + escapeHTML(item.rpcName || "-") + "</td>" +
					"<td>" + String(item.statusCode || 0) + "</td>" +
					"<td>" + String(item.durationMs || 0) + " ms</td>" +
					"<td>" + escapeHTML(item.guardOutcome || "-") + "</td>" +
					"<td class='message' title='" + escapeHTML(item.errorMessage || "-") + "'>" + escapeHTML(item.errorMessage || "-") + "</td>" +
				"</tr>"
			}, "No sampled traces captured yet.")
		}

		function fetchObservability() {
			return fetch(METRICS_URL)
				.then(function (response) {
					if (!response.ok) {
						throw new Error("status " + response.status)
					}
					return response.json()
				})
				.then(function (payload) {
					metricsSnapshot = payload || null
					renderObservability()
				})
				.catch(function () {
					metricsSnapshot = null
					renderObservability()
				})
		}

		function formatFileMeta(file) {
			const tags = []
			tags.push(String(file.lines || 0) + " lines")
			tags.push(formatBytes(file.bytes || 0))
			if (file.truncated) {
				tags.push("truncated")
			}
			return tags.join(" | ")
		}

		function showSQLFile(kind, index) {
			const list = kind === "schemas" ? sqlCatalog.schemas : sqlCatalog.queries
			if (!Array.isArray(list) || list.length === 0 || index < 0 || index >= list.length) {
				return
			}
			selected = { kind: kind, index: index }
			const file = list[index]
			document.getElementById("sql-file-path").textContent = file.path || file.name || "SQL file"
			document.getElementById("sql-file-details").textContent = formatFileMeta(file)
			document.getElementById("sql-content").textContent = file.content || ""
			renderSQLLists()
		}

		function renderSQLList(containerID, kind, items) {
			const node = document.getElementById(containerID)
			node.innerHTML = ""
			if (!Array.isArray(items) || items.length === 0) {
				const empty = document.createElement("div")
				empty.className = "empty"
				empty.textContent = "No SQL files found"
				node.appendChild(empty)
				return
			}
			items.forEach(function (file, index) {
				const button = document.createElement("button")
				button.className = "sql-item"
				if (selected.kind === kind && selected.index === index) {
					button.classList.add("active")
				}
				button.type = "button"
				button.textContent = file.name || file.path || (kind + "-" + String(index + 1))
				const meta = document.createElement("small")
				meta.textContent = formatFileMeta(file)
				button.appendChild(meta)
				button.addEventListener("click", function () {
					showSQLFile(kind, index)
				})
				node.appendChild(button)
			})
		}

		function renderSQLLists() {
			renderSQLList("schemas-list", "schemas", sqlCatalog.schemas)
			renderSQLList("queries-list", "queries", sqlCatalog.queries)
		}

		function renderSQLCatalog(catalog) {
			sqlCatalog = catalog || { schemas: [], queries: [] }
			document.getElementById("sql-root").textContent = sqlCatalog.root || "db/sql"
			const status = document.getElementById("sql-status")
			const layout = document.getElementById("sql-layout")
			if (sqlCatalog.error) {
				status.textContent = "SQL catalog error: " + sqlCatalog.error
				status.hidden = false
				layout.hidden = true
				return
			}
			if (sqlCatalog.missing) {
				status.textContent = "No db/sql folder found from current working directory."
				status.hidden = false
				layout.hidden = true
				return
			}
			status.hidden = true
			layout.hidden = false
			renderSQLLists()
			if ((!selected.kind || selected.kind === "schemas") && Array.isArray(sqlCatalog.schemas) && sqlCatalog.schemas.length > 0) {
				showSQLFile("schemas", Math.min(selected.index, sqlCatalog.schemas.length - 1))
				return
			}
			if (selected.kind === "queries" && Array.isArray(sqlCatalog.queries) && sqlCatalog.queries.length > 0) {
				showSQLFile("queries", Math.min(selected.index, sqlCatalog.queries.length - 1))
				return
			}
			if (Array.isArray(sqlCatalog.queries) && sqlCatalog.queries.length > 0) {
				showSQLFile("queries", 0)
				return
			}
			document.getElementById("sql-file-path").textContent = "No SQL file selected"
			document.getElementById("sql-file-details").textContent = ""
			document.getElementById("sql-content").textContent = ""
		}

		function loadSQLCatalog() {
			fetch(SQL_CATALOG_URL, { headers: { "Accept": "application/json" } })
				.then(function (response) {
					if (!response.ok) {
						throw new Error("status " + response.status)
					}
					return response.json()
				})
				.then(renderSQLCatalog)
				.catch(function (err) {
					document.getElementById("sql-status").textContent = "Failed to load SQL catalog: " + err.message
				})
		}

		function renderLoggingState(status) {
			loggingStatus = status || { enabled: false, active: false, snippet: "" }
			const disabled = document.getElementById("logs-disabled")
			const table = document.getElementById("logs-table-wrap")
			const snippet = document.getElementById("logs-snippet")

			if (!loggingStatus.enabled) {
				disabled.hidden = false
				table.hidden = true
				snippet.textContent = loggingStatus.snippet || "Attach logger with router.AttachLogger(...)"
				setStreamState("disabled")
				return
			}

			disabled.hidden = true
			table.hidden = false
			if (loggingStatus.active) {
				setStreamState("live")
			} else {
				setStreamState("idle")
			}
		}

		function loadLoggingStatus() {
			return fetch(LOGGING_STATUS_URL, { headers: { "Accept": "application/json" } })
				.then(function (response) {
					if (!response.ok) {
						throw new Error("status " + response.status)
					}
					return response.json()
				})
				.then(function (payload) {
					renderLoggingState(payload)
				})
				.catch(function () {
					renderLoggingState({ enabled: false, active: false, snippet: "" })
				})
		}

		function loadEventSnapshot() {
			return fetch(EVENTS_URL + "?limit=200", { headers: { "Accept": "application/json" } })
				.then(function (response) {
					if (!response.ok) {
						throw new Error("status " + response.status)
					}
					return response.json()
				})
				.then(function (payload) {
					if (payload && Array.isArray(payload.events)) {
						applyEvents(payload.events)
					}
				})
		}

		function startPollingFallback() {
			setStreamState("polling")
			window.setInterval(function () {
				loadEventSnapshot().catch(function () {})
			}, 4000)
		}

		function startEventStream() {
			if (typeof EventSource === "undefined") {
				startPollingFallback()
				return
			}
			setStreamState("connecting")
			eventSource = new EventSource(EVENTS_STREAM_URL)
			eventSource.onopen = function () {
				streamConnected = true
				setStreamState("live")
			}
			eventSource.onmessage = function (evt) {
				try {
					const item = JSON.parse(evt.data)
					applyEvents([item])
				} catch (_) {
				}
			}
			eventSource.onerror = function () {
				if (eventSource) {
					eventSource.close()
					eventSource = null
				}
				if (reconnectTimer) {
					window.clearTimeout(reconnectTimer)
				}
				setStreamState(streamConnected ? "reconnecting" : "offline")
				reconnectTimer = window.setTimeout(function () {
					loadEventSnapshot().catch(function () {})
					startEventStream()
				}, 2000)
			}
		}

		showPanel(activePanelFromHash())
		mountScalar()
		fetchObservability()
		window.setInterval(fetchObservability, 5000)
		loadSQLCatalog()
		loadLoggingStatus().then(function () {
			if (!loggingStatus.enabled) {
				return
			}
			loadEventSnapshot().finally(function () {
				startEventStream()
			})
		})
	})()
	</script>
</body>
</html>
`
