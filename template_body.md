## Go-Healthier Report
{{ range $item := . }}
### Result {{ $item.Request.Url }}

* **Request Config**
  * url: {{ $item.Request.Url }}
  * method: {{ $item.Request.Method }}

* **Request Result**
  * status: {{ if $item.IsFailed }} :no_good: Failed {{ else if not $item.IsSucceed }} :red_circle: Error {{ else }} :green_circle: OK {{ end }}
  * HTTP Status Code: {{ $item.StatusCode }}
{{ if $item.IsSucceed }}  * Duration: {{ $item.Duration }}
{{ else }}  * Error Message: {{ $item.ErrorMsg }}{{ end }}
{{ end }}