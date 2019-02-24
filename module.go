package jsonrpc

import (
	"github.com/im-kulikov/helium/module"
)

// Module for helium json-rpc
var Module = module.Module{
	{Constructor: NewRPC},
}
