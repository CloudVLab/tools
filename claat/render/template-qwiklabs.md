# {{.Meta.Title}}

{{range .Steps}}{{if matchEnv .Tags $.Env}}
# {{.Title}}

{{if .Duration}}*Duration is {{.Duration.Minutes}} min*{{end}}
{{.Content | renderQwiklabs $.Env}}
{{end}}{{end}}

{{if .Meta.Feedback}}[Provide Feedback on this Lab]({{.Meta.Feedback}}){{end}}

