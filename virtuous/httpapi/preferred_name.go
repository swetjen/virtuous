package httpapi

import (
	"reflect"

	"github.com/swetjen/virtuous/schema"
)

func preferredSchemaName(meta HandlerMeta, t reflect.Type) string {
	return schema.PreferredNameOf(meta.Service, t)
}
