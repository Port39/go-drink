package handlehttp

import (
	"html/template"
	"io/fs"
	"log"
	"net/http"
)

func HtmlMapper(tplFS fs.FS, tplPaths ...string) ResponseMapper {
	tpl := template.Must(template.ParseFS(tplFS, tplPaths...))

	return func(w http.ResponseWriter, status int, data any) {
		w.Header().Set("content-type", Html.String())
		w.WriteHeader(status)
		err := tpl.Execute(w, tpl)

		if err != nil {
			log.Println("Error while writing html response:", err)
			return
		}
	}
}
