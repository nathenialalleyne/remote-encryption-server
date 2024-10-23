package handlers

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/nathenialalleyne/remote-encryption-service/pkg/helpers"
)

func EncryptionHandler() func(http.ResponseWriter, *http.Request) {

    return func(w http.ResponseWriter, r *http.Request) {
		
		cmd, err := spawnEncryptionProcess()

		if err != nil{
			helpers.HandleError(err)
		}

		output, err := cmd.Output()

		if err != nil{
			helpers.HandleError(err)
		}

		fmt.Println(output)

        w.WriteHeader(http.StatusOK)

    }
}

func spawnEncryptionProcess() (*exec.Cmd, error) {
	cmd := exec.Command("go", "run", "../../internal/services/encryption_service.go")
	
	var errBuf, outBuf bytes.Buffer
	cmd.Stderr = io.MultiWriter(os.Stderr, &errBuf)
	cmd.Stdout = io.MultiWriter(os.Stdout, &outBuf)

	err := cmd.Start()
	if err != nil {
		return nil, err
	}

	fmt.Printf("Encryption service started with PID: %d\n", cmd.Process.Pid)
	time.Sleep(2 * time.Second)

	return cmd, nil
}