package services

import (
	"os/exec"

	"github.com/nathenialalleyne/remote-encryption-service/pkg/helpers"
)

func EncryptionService() []byte {
	cmd := exec.Command("go", "run", "../services/encryption_service.go")

	output, err := cmd.Output()

	if err != nil{
		helpers.HandleError(err)
	}

	return output
}