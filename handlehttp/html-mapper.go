package handlehttp

import (
	"html/template"
	"io/fs"
	"log"
	"net/http"
)

func HtmlMapper(tplFS fs.FS, tplPaths ...string) ResponseMapper {
	templates := append(tplPaths, "base-templates/page-layout.gohtml")
	tpl := template.Must(template.ParseFS(tplFS, templates...))

	return func(w http.ResponseWriter, status int, data any) {
		w.Header().Set("content-type", Html.String())
		w.WriteHeader(status)
		err := executeWholePageTemplate(w, tpl, data)

		if err != nil {
			log.Println("Error while writing html response:", err)
			return
		}
	}
}

func executeWholePageTemplate(w http.ResponseWriter, t *template.Template, data interface{}) error {
	return t.ExecuteTemplate(w, "base", data)
}

func executeFragmentTemplate(w http.ResponseWriter, t *template.Template, data interface{}) error {
	return t.ExecuteTemplate(w, "fragment", data)
}
