package handlehttp

import (
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"reflect"
)

func hasField(v interface{}, name string) bool {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return false
	}
	return rv.FieldByName(name).IsValid()
}

func HtmlMapper(tplFS fs.FS, useFragment bool, tplPaths ...string) ResponseMapper {
	templates := append(tplPaths, "base-templates/page-layout.gohtml", "base-templates/fragment.gohtml")
	tpl := template.Must(template.New("template").Funcs(template.FuncMap{
		"hasField": hasField,
	}).ParseFS(tplFS, templates...))

	return func(w http.ResponseWriter, status int, data any) {
		w.Header().Set("content-type", Html.String())
		w.WriteHeader(status)
		var err error
		if useFragment {
			err = executeFragmentTemplate(w, tpl, data)
		} else {
			err = executeWholePageTemplate(w, tpl, data)
		}

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
