package main

import (
	"html/template"
)

type TemplateData struct {
	Store        Store
	CsrfTemplate template.HTML
	Errors       []error
}

var templates = template.Must(template.New("form.html").Parse(
`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>RTMP Admin</title>
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <link rel="stylesheet" type="text/css" href="{{.Store.Prefix}}/public/mini-dark.css">
  <link rel="stylesheet" type="text/css" href="{{.Store.Prefix}}/public/main.css">
</head>
<body>
  <div class="container">
    <h1><a href="/">rtmp-auth</a></h1>
    <h2>Streams</h2>

    <div class="row">
      {{range .Errors}}
        <div class="card error">
          <div class="section">
            <h3>Error</h3>
            <p>{{.Error}}</p>
          </div>
        </div>
      {{end}}
    </div>

    <table>
      <thead>
        <th>Name</th>
        <th data-label="Auth">Auth</th>
        <th>Expires</th>
        <th data-label="Notes">Notes</th>
        <th></th>
      </thead>
      <tbody>
      {{range .Store.State.Streams}}
        <tr>
          <td data-label="Name">
            {{.Application}}/{{.Name}}
            {{if .Active}}
              <mark class="tag">live</mark>
            {{end}}
          </td>
          <td data-label="Auth">
            <input class="authKey" size="5" value="{{.AuthKey}}" readonly/><button class="secondary copyToClipboard inputAddon">Copy</button>
          </td>
          <td data-label="Expire" data-expire="{{.AuthExpire}}">
            {{if eq .AuthExpire -1}}
              never
            {{else}}
              {{.AuthExpire}}
            {{end}}
          </td>
          <td data-label="Notes">{{.Notes}}</td>
          <td style="text-align:right;">
            <form class="inline" action="{{$.Store.Prefix}}/remove" method="POST">
              {{ $.CsrfTemplate }}
              <input type="hidden" name="id" value="{{.Id}}">
              <button class="secondary">Remove</button>
            </form>
          </td>
        </tr>
      {{end}}
      </tbody>
    </table>

    <h2>Add Stream</h2>
    <form class="addForm" action="{{$.Store.Prefix}}/add" method="POST" novalidate>
      <div class="row">
        <div class="col-sm-12 col-md-6">
          <label for="application">Application</label>
          <select type="text" id="application" name="application">
            {{range .Store.Applications}}
              <option value="{{.}}">{{.}}</option>
            {{end}}
          </select>
        </div>

        <div class="col-sm-12 col-md-6">
          <label for="stream">Stream</label>
          <input type="text" size="5" id="stream" name="name" placeholder="enter name">
        </div>

        <div class="col-sm-12 col-md-6">
          <label for="authKey">Auth Key</label>
          <input type="text" size="3" id="authKey" name="auth_key" placeholder="no auth"><button class="secondary generateKey inputAddon">Generate key</button>
        </div>

        <div class="col-sm-12 col-md-6">
          <label for="authExpire">Auth Expire
            <span class="tooltip" aria-label="ISO8601 Duration (e.g. P2DT10H) or empty for no expiry">
              <span class="icon-help"></span>
            </span>
          </label>
          <input type="text" size="5" id="authExpire" name="auth_expire" placeholder="never">
        </div>

        <div class="col-sm-12">
          <label for="notes">Notes</label>
          <input type="text" size="5" id="notes" name="notes" placeholder="optional notes">
        </div>
      </div>

      <div class="row">
        {{ .CsrfTemplate }}
        <div class="col-sm-12 col-md-12">
          <button class="primary">Submit</button>
        </div>
      </div>
    </form>
  </div>
<script src="{{.Store.Prefix}}/public/main.js"></script>
</body>
</html>`))
