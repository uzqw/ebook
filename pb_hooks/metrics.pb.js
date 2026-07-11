// Minimal Prometheus-compatible metrics for the ebook reader app.
// Keep this endpoint cheap: it is scraped frequently by VictoriaMetrics.

routerAdd("GET", "/metrics", function (e) {
  var now = Math.floor(Date.now() / 1000)
  var lines = [
    "# HELP ebook_reader_up Whether the ebook reader PocketBase process is serving requests.",
    "# TYPE ebook_reader_up gauge",
    "ebook_reader_up{app=\"ebook_reader_uzqw\"} 1",
    "# HELP ebook_reader_metrics_timestamp_seconds Unix timestamp when this scrape response was generated.",
    "# TYPE ebook_reader_metrics_timestamp_seconds gauge",
    "ebook_reader_metrics_timestamp_seconds{app=\"ebook_reader_uzqw\"} " + now,
    ""
  ]

  e.string(200, lines.join("\n"))
})
