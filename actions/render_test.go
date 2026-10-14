package actions

import (
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"net/http"
	"todo/public"
)

// Los assets se sirven con caché de un año: la URL tiene que cambiar cuando
// cambia su contenido, o el navegador sigue usando la versión anterior.
func (as *ActionSuite) Test_Layout_AssetURLsCarryContentHash() {
	res := as.HTML("/").Get()
	as.Equal(http.StatusOK, res.Code)

	for _, name := range []string{"app.css", "app.js"} {
		content, err := fs.ReadFile(public.FS(), "assets/"+name)
		as.NoError(err)
		sum := sha256.Sum256(content)
		as.Contains(res.Body.String(), "/assets/"+name+"?v="+hex.EncodeToString(sum[:])[:12])
	}
}
