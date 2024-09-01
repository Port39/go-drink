package handlehttp

import (
	"html/template"
	"io/fs"
	"log"
	"net/http"
)

func activateHtmlResponse(w http.ResponseWriter) {
	w.Header().Set("content-type", Html.String())
}

func HtmlMapper(tplFS fs.FS, tplPaths ...string) ResponseMapper {
	tpl := template.Must(template.ParseFS(tplFS, tplPaths...))

	return func(w http.ResponseWriter, data any) {
		err := tpl.Execute(w, tpl)

		if err != nil {
			log.Println("Error while writing html response:", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
}
