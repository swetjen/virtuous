package virtuous

import "github.com/swetjen/virtuous/rpc"

// RPC type aliases for convenience.
type RPCGuard = rpc.Guard
type RPCGuardSpec = rpc.GuardSpec
type RPCRoute = rpc.Route
type RPCRouter = rpc.Router
type RPCResult[Ok, Err any] = rpc.Result[Ok, Err]
type RPCTypeOverride = rpc.TypeOverride

type RPCDocsOptions = rpc.DocsOptions
type RPCDocOpt = rpc.DocOpt
type RPCServeAllDocsOptions = rpc.ServeAllDocsOptions
type RPCServeAllDocsOpt = rpc.ServeAllDocsOpt

type RPCOpenAPIOptions = rpc.OpenAPIOptions
type RPCOpenAPIServer = rpc.OpenAPIServer
type RPCOpenAPITag = rpc.OpenAPITag
type RPCOpenAPIContact = rpc.OpenAPIContact
type RPCOpenAPILicense = rpc.OpenAPILicense
type RPCOpenAPIExternalDocs = rpc.OpenAPIExternalDocs

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

func RPCOK[Ok, Err any](v Ok) rpc.Result[Ok, Err] {
	return rpc.OK[Ok, Err](v)
}

func RPCInvalid[Ok, Err any](e Err) rpc.Result[Ok, Err] {
	return rpc.Invalid[Ok, Err](e)
}

func RPCFail[Ok, Err any](e Err) rpc.Result[Ok, Err] {
	return rpc.Fail[Ok, Err](e)
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
