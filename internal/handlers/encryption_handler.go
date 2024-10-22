package handlers

import (
	"net/http"

	"github.com/nathenialalleyne/remote-encryption-service/internal/services"
)

func EncryptionHandler() func(http.ResponseWriter, *http.Request) {

    return func(w http.ResponseWriter, r *http.Request) {
		
		

        w.WriteHeader(http.StatusOK)

        w.Write(services.EncryptionService())

    }
}