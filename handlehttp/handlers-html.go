package handlehttp

import (
	"embed"
	"html/template"
	"log"
	"net/http"
)

//go:embed html/*.gohtml
var tplFS embed.FS

func activateHtmlResponse(w http.ResponseWriter) {
	w.Header().Set("content-type", Html.String())
}

func HtmlMapper(tplPaths ...string) ResponseMapper {
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
