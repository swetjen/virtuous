package byodbsqlite

import (
	"os"
	"path/filepath"

	"github.com/swetjen/virtuous/rpc"
)

const frontendClientPath = "static/client.gen.js"

// WriteFrontendClient writes the generated JS client to the frontend api folder.
func WriteFrontendClient(router *rpc.Router) error {
	if router == nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(frontendClientPath), 0755); err != nil {
		return err
	}
	return router.WriteClientJSFile(frontendClientPath)
}
