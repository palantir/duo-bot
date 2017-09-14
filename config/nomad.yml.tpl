{{with secret "secret/nomad/duo-bot/duo"}}
duo:
  host: "{{.Data.host}}"
  ikey: "{{.Data.ikey}}"
  skey: "{{.Data.skey}}"
{{end}}
