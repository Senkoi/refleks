// Package contracts generates JSON TypeScript contracts from Go DTOs.
package contracts

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"aimmeow/internal/constants"
	"aimmeow/internal/practice"
	"aimmeow/internal/training"
)

func Generate() []byte {
	roots := []reflect.Type{reflect.TypeOf(training.TrainingProgressDTO{}), reflect.TypeOf(practice.SessionGroup{}), reflect.TypeOf(training.WorkbenchDTO{}), reflect.TypeOf(training.LiveState{}), reflect.TypeOf(training.GenerateRequest{}), reflect.TypeOf(training.ScenarioComparison{}), reflect.TypeOf(training.PlanDefinition{}), reflect.TypeOf(training.ExecutionProgress{})}
	types := map[string]reflect.Type{}
	var visit func(reflect.Type)
	visit = func(t reflect.Type) {
		switch t.Kind() {
		case reflect.Pointer, reflect.Slice, reflect.Array:
			visit(t.Elem())
		case reflect.Map:
			visit(t.Elem())
		case reflect.Struct:
			if _, ok := types[t.Name()]; ok {
				return
			}
			types[t.Name()] = t
			for i := 0; i < t.NumField(); i++ {
				f := t.Field(i)
				if f.IsExported() && f.Tag.Get("json") != "-" {
					visit(f.Type)
				}
			}
		}
	}
	for _, t := range roots {
		visit(t)
	}
	var tsType func(reflect.Type) string
	tsType = func(t reflect.Type) string {
		switch t.Kind() {
		case reflect.Pointer:
			return tsType(t.Elem()) + " | null"
		case reflect.String:
			return "string"
		case reflect.Bool:
			return "boolean"
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Float32, reflect.Float64:
			return "number"
		case reflect.Slice:
			return "Array<" + tsType(t.Elem()) + ">"
		case reflect.Array:
			items := []string{}
			for i := 0; i < t.Len(); i++ {
				items = append(items, tsType(t.Elem()))
			}
			return "[" + strings.Join(items, ", ") + "]"
		case reflect.Map:
			return "Record<string, " + tsType(t.Elem()) + ">"
		case reflect.Struct:
			return t.Name()
		}
		panic(fmt.Sprintf("unsupported contract type %s", t))
	}
	names := []string{}
	for name := range types {
		names = append(names, name)
	}
	sort.Strings(names)
	var b strings.Builder
	b.WriteString("// Generated from Go training DTOs. DO NOT EDIT.\n// Regenerate: go run ./cmd/training-contracts\n\n")
	fmt.Fprintf(&b, "export const DEFAULT_SESSION_GAP_MINUTES = %d;\n\n", constants.DefaultSessionGapMinutes)
	for _, name := range names {
		t := types[name]
		fmt.Fprintf(&b, "export type %s = {\n", name)
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if !f.IsExported() {
				continue
			}
			tag := strings.Split(f.Tag.Get("json"), ",")
			if tag[0] == "-" {
				continue
			}
			key := tag[0]
			if key == "" {
				key = f.Name
			}
			optional := ""
			for _, option := range tag[1:] {
				if option == "omitempty" {
					optional = "?"
				}
			}
			fieldType := tsType(f.Type)
			if optional != "" && f.Type.Kind() == reflect.Pointer {
				fieldType = tsType(f.Type.Elem())
			}
			fmt.Fprintf(&b, "  %s%s: %s;\n", key, optional, fieldType)
		}
		b.WriteString("};\n\n")
	}
	return []byte(strings.TrimRight(b.String(), "\n") + "\n")
}
