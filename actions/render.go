package actions

import (
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"path"
	"sync"
	"todo/public"
	"todo/templates"

	"github.com/gobuffalo/buffalo/render"
)

var r *render.Engine

func init() {
	r = render.New(render.Options{
		// HTML layout to be used for all HTML requests:
		HTMLLayout: "application.plush.html",

		// fs.FS containing templates
		TemplatesFS: templates.FS(),

		// fs.FS containing assets
		AssetsFS: public.FS(),

		// Add template helpers here:
		Helpers: render.Helpers{
			"assetURL": assetURL,
			// for non-bootstrap form helpers uncomment the lines
			// below and import "github.com/gobuffalo/helpers/forms"
			// forms.FormKey:     forms.Form,
			// forms.FormForKey:  forms.FormFor,
		},
	})
}

var assetVersions sync.Map

// assetURL devuelve la URL de un fichero de public/assets con un hash de su
// contenido. Los assets se sirven con caché de un año, así que un número de
// versión manual deja al navegador con ficheros antiguos (o de otra app que
// usara la misma URL en localhost) en cuanto alguien olvida subirlo.
func assetURL(name string) string {
	url := "/assets/" + name
	if version, ok := assetVersions.Load(name); ok {
		return url + "?v=" + version.(string)
	}

	content, err := fs.ReadFile(public.FS(), path.Join("assets", name))
	if err != nil {
		return url
	}
	sum := sha256.Sum256(content)
	version := hex.EncodeToString(sum[:])[:12]
	assetVersions.Store(name, version)
	return url + "?v=" + version
}
