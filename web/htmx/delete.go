package htmx

import (
	"net/http"

	"github.com/hatmaxkit/hatmax/log"
)

func RespondDelete(w http.ResponseWriter, err error, log log.Logger) {
	if err != nil {
		log.Errorf("error deleting: %v", err)
		http.Error(w, "Cannot delete", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(""))
}
