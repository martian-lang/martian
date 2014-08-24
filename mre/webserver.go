//
// Copyright (c) 2014 10X Technologies, Inc. All rights reserved.
//
// mre webserver.
//
package main

import (
	"bytes"
	"encoding/json"
	"github.com/go-martini/martini"
	"github.com/martini-contrib/binding"
	"html/template"
	"io/ioutil"
	"margo/core"
	"net/http"
	"path"
	"path/filepath"
	"regexp"
)

// Render JSON from data.
func makeJSON(data interface{}) string {
	bytes, err := json.Marshal(data)
	if err != nil {
		return err.Error()
	}
	return string(bytes)
}

type LoadForm struct {
	Fname string
}

type SaveForm struct {
	Fname    string
	Contents string
}

func runWebServer(uiport string, rt *core.Runtime) {
	//=========================================================================
	// Configure server.
	//=========================================================================
	m := martini.New()
	r := martini.NewRouter()
	m.Use(martini.Recovery())
	m.Use(martini.Static("../web/res", martini.StaticOptions{"", true, "index.html", nil}))
	m.Use(martini.Static("../web/client", martini.StaticOptions{"", true, "index.html", nil}))
	m.MapTo(r, (*martini.Routes)(nil))
	m.Action(r.Handle)
	app := &martini.ClassicMartini{m, r}

	//=========================================================================
	// Page renderers.
	//=========================================================================
	app.Get("/", func() string {
		tmpl, _ := template.New("editor.html").Delims("[[", "]]").ParseFiles("../web/templates/editor.html")
		var doc bytes.Buffer
		tmpl.Execute(&doc, map[string]interface{}{})
		return doc.String()
	})

	//=========================================================================
	// API endpoints.
	//=========================================================================

	// Get list of names of MRO files in the runtime's MRO path.
	app.Get("/files", func() string {
		filePaths, _ := filepath.Glob(path.Join(rt.MroPath, "*"))
		fnames := []string{}
		for _, filePath := range filePaths {
			fnames = append(fnames, filepath.Base(filePath))
		}
		return makeJSON(fnames)
	})

	// Load the contents of the specified MRO file plus the contents
	// of its first included file for the 2-up view.
	re := regexp.MustCompile("@include \"([^\"]+)")
	app.Post("/load", binding.Bind(LoadForm{}), func(body LoadForm, p martini.Params) string {
		// Load contents of selected file.
		bytes, _ := ioutil.ReadFile(path.Join(rt.MroPath, body.Fname))
		contents := string(bytes)

		// Parse the first @include line.
		submatches := re.FindStringSubmatch(contents)
		if len(submatches) > 1 {

			// Load contents of included file.
			includeFname := submatches[1]
			includeBytes, _ := ioutil.ReadFile(path.Join(rt.MroPath, includeFname))
			return makeJSON(map[string]string{
				"contents":        contents,
				"includeFname":    includeFname,
				"includeContents": string(includeBytes),
			})
		}
		return makeJSON(map[string]interface{}{"contents": contents})
	})

	// Save file.
	app.Post("/save", binding.Bind(SaveForm{}), func(body SaveForm, p martini.Params) string {
		ioutil.WriteFile(path.Join(rt.MroPath, body.Fname), []byte(body.Contents), 0600)
		return ""
	})

	// Compile file.
	app.Post("/build", binding.Bind(LoadForm{}), func(body LoadForm, p martini.Params) string {
		global, err := rt.Compile(body.Fname)
		if err != nil {
			return err.Error()
		}
		return makeJSON(global)
	})

	//=========================================================================
	// Start webserver.
	//=========================================================================
	http.ListenAndServe(":"+uiport, app)
}
