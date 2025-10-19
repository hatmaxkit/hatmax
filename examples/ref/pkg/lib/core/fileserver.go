package core

import (
	"embed"
	"io/fs"
	"net/http"

	"github.com/go-chi/chi/v5"
)

const (
	staticAssetsPath = "assets/static"
	staticURLPrefix  = "/static"
)

type FileServer struct {
	log      Logger
	assetsFS embed.FS
}

func NewFileServer(assetsFS embed.FS, log Logger) *FileServer {
	return &FileServer{
		log:      log,
		assetsFS: assetsFS,
	}
}

func (s *FileServer) RegisterRoutes(r chi.Router) {
	s.log.Info("Registering file server", "url", staticURLPrefix, "dir", staticAssetsPath)

	staticFS, err := fs.Sub(s.assetsFS, staticAssetsPath)
	if err != nil {
		s.log.Error("error creating static files sub-filesystem", "error", err)
		return
	}

	r.Handle(staticURLPrefix+"/*", http.StripPrefix(staticURLPrefix+"/", http.FileServer(http.FS(staticFS))))
	s.log.Info("File server registered successfully")
}
