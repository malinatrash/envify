package parser

import (
	"fmt"
	"go/types"
	"reflect"
	"strings"

	"golang.org/x/tools/go/packages"
)

type EnvEntry struct {
	Key        string
	Default    string
	Required   bool
	Masked     bool
	GroupBreak bool
}

func Extract(dir, typeName string) ([]EnvEntry, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedSyntax | packages.NeedDeps | packages.NeedImports,
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
		extractFromStruct(st, "", &entries, true)
		return entries, nil
	}

	return nil, fmt.Errorf("type %s not found in package at %s", typeName, dir)
}

func extractFromStruct(st *types.Struct, prefix string, entries *[]EnvEntry, _ bool) {
	for i := range st.NumFields() {
		field := st.Field(i)
		tag := reflect.StructTag(st.Tag(i))

		envTag := tag.Get("env")
		envPrefix := tag.Get("envPrefix")
		envDefault := tag.Get("envDefault")

		ft := field.Type()
		if ptr, ok := ft.(*types.Pointer); ok {
			ft = ptr.Elem()
		}

		if nested, ok := ft.Underlying().(*types.Struct); ok && envTag == "" {
			groupBreak := len(*entries) > 0
			sub := collectEntries(nested, prefix+envPrefix)
			if len(sub) > 0 {
				sub[0].GroupBreak = groupBreak
			}
			*entries = append(*entries, sub...)
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

func collectEntries(st *types.Struct, prefix string) []EnvEntry {
	var entries []EnvEntry
	extractFromStruct(st, prefix, &entries, false)
	return entries
}
