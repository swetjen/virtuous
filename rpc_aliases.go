package virtuous

import "github.com/swetjen/virtuous/rpc"

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

const (
	RPCModuleAPI           = rpc.ModuleAPI
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

func RPCWithMaxRequestBodyBytes(maxBytes int64) rpc.RouterOption {
	return rpc.WithMaxRequestBodyBytes(maxBytes)
}

func RPCWithAdvancedObservability(opts ...rpc.AdvancedObservabilityOption) rpc.RouterOption {
	return rpc.WithAdvancedObservability(opts...)
}

func RPCWithObservabilitySampling(rate float64) rpc.AdvancedObservabilityOption {
	return rpc.WithObservabilitySampling(rate)
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

func RPCWithDocsGuards(guards ...rpc.Guard) RPCDocOpt {
	return rpc.WithDocsGuards(guards...)
}

func RPCWithAdminGuards(guards ...rpc.Guard) RPCDocOpt {
	return rpc.WithAdminGuards(guards...)
}

func RPCWithPublicAdmin() RPCDocOpt {
	return rpc.WithPublicAdmin()
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
