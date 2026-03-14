package virtuous

import (
	"database/sql"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/swetjen/virtuous/rpc"
)

// RPC type aliases for convenience.
type RPCGuard = rpc.Guard
type RPCGuardSpec = rpc.GuardSpec
type RPCRoute = rpc.Route
type RPCRouter = rpc.Router
type RPCTypeOverride = rpc.TypeOverride

type RPCDocsOptions = rpc.DocsOptions
type RPCDocOpt = rpc.DocOpt
type RPCModule = rpc.Module
type RPCServeAllDocsOptions = rpc.ServeAllDocsOptions
type RPCServeAllDocsOpt = rpc.ServeAllDocsOpt

type RPCOpenAPIOptions = rpc.OpenAPIOptions
type RPCOpenAPIServer = rpc.OpenAPIServer
type RPCOpenAPITag = rpc.OpenAPITag
type RPCOpenAPIContact = rpc.OpenAPIContact
type RPCOpenAPILicense = rpc.OpenAPILicense
type RPCOpenAPIExternalDocs = rpc.OpenAPIExternalDocs
type RPCAdvancedObservabilityOptions = rpc.AdvancedObservabilityOptions
type RPCAdvancedObservabilityOption = rpc.AdvancedObservabilityOption
type RPCDBExplorer = rpc.DBExplorer
type RPCDBExplorerOptions = rpc.DBExplorerOptions
type RPCDBExplorerOption = rpc.DBExplorerOption
type RPCDBExplorerState = rpc.DBExplorerState
type RPCDBSchema = rpc.DBSchema
type RPCDBTable = rpc.DBTable
type RPCDBPreviewInput = rpc.DBPreviewInput
type RPCDBRunQueryInput = rpc.DBRunQueryInput
type RPCDBQueryResult = rpc.DBQueryResult
type RPCDBQueryColumn = rpc.DBQueryColumn

const (
	RPCModuleAPI           = rpc.ModuleAPI
	RPCModuleDatabase      = rpc.ModuleDatabase
	RPCModuleObservability = rpc.ModuleObservability

	RPCStatusOK      = rpc.StatusOK
	RPCStatusInvalid = rpc.StatusInvalid
	RPCStatusError   = rpc.StatusError
)

// RPC function shims.
func NewRPCRouter(opts ...rpc.RouterOption) *rpc.Router {
	return rpc.NewRouter(opts...)
}

func RPCWithPrefix(prefix string) rpc.RouterOption {
	return rpc.WithPrefix(prefix)
}

func RPCWithGuards(guards ...rpc.Guard) rpc.RouterOption {
	return rpc.WithGuards(guards...)
}

func RPCWithAdvancedObservability(opts ...rpc.AdvancedObservabilityOption) rpc.RouterOption {
	return rpc.WithAdvancedObservability(opts...)
}

func RPCWithObservabilitySampling(rate float64) rpc.AdvancedObservabilityOption {
	return rpc.WithObservabilitySampling(rate)
}

func RPCWithDBExplorer(explorer rpc.DBExplorer) rpc.RouterOption {
	return rpc.WithDBExplorer(explorer)
}

func RPCWithDBExplorerTimeout(timeout time.Duration) rpc.DBExplorerOption {
	return rpc.WithDBExplorerTimeout(timeout)
}

func RPCWithDBExplorerMaxRows(maxRows int) rpc.DBExplorerOption {
	return rpc.WithDBExplorerMaxRows(maxRows)
}

func RPCWithDBExplorerPreviewRows(previewRows int) rpc.DBExplorerOption {
	return rpc.WithDBExplorerPreviewRows(previewRows)
}

func RPCWithDBExplorerSystemSchemas(enabled bool) rpc.DBExplorerOption {
	return rpc.WithDBExplorerSystemSchemas(enabled)
}

func RPCNewSQLDBExplorer(db *sql.DB, opts ...rpc.DBExplorerOption) rpc.DBExplorer {
	return rpc.NewSQLDBExplorer(db, opts...)
}

func RPCNewPGXDBExplorer(pool *pgxpool.Pool, opts ...rpc.DBExplorerOption) rpc.DBExplorer {
	return rpc.NewPGXDBExplorer(pool, opts...)
}

func RPCDefaultDocsHTML(openAPIPath string) string {
	return rpc.DefaultDocsHTML(openAPIPath)
}

func RPCWriteDocsHTMLFile(path, openAPIPath string) error {
	return rpc.WriteDocsHTMLFile(path, openAPIPath)
}

func RPCWithDocsPath(path string) RPCDocOpt {
	return rpc.WithDocsPath(path)
}

func RPCWithDocsFile(path string) RPCDocOpt {
	return rpc.WithDocsFile(path)
}

func RPCWithOpenAPIPath(path string) RPCDocOpt {
	return rpc.WithOpenAPIPath(path)
}

func RPCWithOpenAPIFile(path string) RPCDocOpt {
	return rpc.WithOpenAPIFile(path)
}

func RPCWithModules(modules ...rpc.Module) RPCDocOpt {
	return rpc.WithModules(modules...)
}

func RPCWithDocsOptions(opts ...rpc.DocOpt) RPCServeAllDocsOpt {
	return rpc.WithDocsOptions(opts...)
}

func RPCWithClientJSPath(path string) RPCServeAllDocsOpt {
	return rpc.WithClientJSPath(path)
}

func RPCWithClientTSPath(path string) RPCServeAllDocsOpt {
	return rpc.WithClientTSPath(path)
}

func RPCWithClientPYPath(path string) RPCServeAllDocsOpt {
	return rpc.WithClientPYPath(path)
}

func RPCWithoutDocs() RPCServeAllDocsOpt {
	return rpc.WithoutDocs()
}
