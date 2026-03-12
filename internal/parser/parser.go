package parser

import (
	"fmt"
	"go/types"
	"reflect"
	"strings"

	"golang.org/x/tools/go/packages"
)

type EnvEntry struct {
	Key      string
	Default  string
	Required bool
	Masked   bool
}

func Extract(dir, typeName string) ([]EnvEntry, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedSyntax,
		Dir:  dir,
	}

	pkgs, err := packages.Load(cfg, ".")
	if err != nil {
		return nil, fmt.Errorf("load packages: %w", err)
	}

	for _, pkg := range pkgs {
		if pkg.Types == nil {
			continue
		}
		obj := pkg.Types.Scope().Lookup(typeName)
		if obj == nil {
			continue
		}
		named, ok := obj.Type().(*types.Named)
		if !ok {
			return nil, fmt.Errorf("%s is not a named type", typeName)
		}
		st, ok := named.Underlying().(*types.Struct)
		if !ok {
			return nil, fmt.Errorf("%s is not a struct", typeName)
		}

		var entries []EnvEntry
		extractFromStruct(st, "", &entries)
		return entries, nil
	}

	return nil, fmt.Errorf("type %s not found in package at %s", typeName, dir)
}

func extractFromStruct(st *types.Struct, prefix string, entries *[]EnvEntry) {
	for i := range st.NumFields() {
		field := st.Field(i)
		tag := reflect.StructTag(st.Tag(i))

		envTag := tag.Get("env")
		envPrefix := tag.Get("envPrefix")
		envDefault := tag.Get("envDefault")

		// Resolve underlying type (deref pointer if needed)
		ft := field.Type()
		if ptr, ok := ft.(*types.Pointer); ok {
			ft = ptr.Elem()
		}

		// Nested struct (no env tag on the field itself)
		if nested, ok := ft.Underlying().(*types.Struct); ok && envTag == "" {
			extractFromStruct(nested, prefix+envPrefix, entries)
			continue
		}

		if envTag == "" {
			continue
		}

		parts := strings.SplitN(envTag, ",", 2)
		key := prefix + parts[0]

		required := len(parts) > 1 && strings.Contains(parts[1], "required")
		masked := tag.Get("print") == "mask"

		*entries = append(*entries, EnvEntry{
			Key:      key,
			Default:  envDefault,
			Required: required,
			Masked:   masked,
		})
	}
}
