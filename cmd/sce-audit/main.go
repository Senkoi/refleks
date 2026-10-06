// sce-audit uses the same parser and descriptor as the desktop application.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"aimmeow/internal/sceneanalysis"
)

type result struct {
	File       string                         `json:"file"`
	Name       string                         `json:"name,omitempty"`
	Assessment *sceneanalysis.LocalAssessment `json:"assessment,omitempty"`
	Error      string                         `json:"error,omitempty"`
}

func read(path string) ([]byte, error) {
	f, e := os.Open(path)
	if e != nil {
		return nil, e
	}
	defer f.Close()
	return io.ReadAll(io.LimitReader(f, sceneanalysis.MaxSCEBytes+1))
}
func main() {
	compare := flag.Bool("compare", false, "Compare exactly two SCE files with fixed-basis gates")
	modelPath := flag.String("model", "", "Optional offline shadow model JSON (requires -compare)")
	flag.Parse()
	paths := flag.Args()
	if len(paths) == 0 || *compare && len(paths) != 2 || *modelPath != "" && !*compare {
		fmt.Fprintln(os.Stderr, "usage: sce-audit [-compare] file.sce [file.sce ...]")
		os.Exit(2)
	}
	var model *sceneanalysis.RankModel
	if *modelPath != "" {
		raw, err := read(*modelPath)
		if err == nil {
			model = &sceneanalysis.RankModel{}
			err = json.Unmarshal(raw, model)
		}
		if err == nil {
			err = model.Validate()
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
	}
	rows := []result{}
	failed := false
	for _, path := range paths {
		r := result{File: path}
		body, e := read(path)
		if e == nil {
			r.Name, _, r.Assessment, e = sceneanalysis.ParseLocalSCE(body)
		}
		if e != nil {
			r.Error = e.Error()
			failed = true
		}
		rows = append(rows, r)
	}
	var value interface{} = rows
	if *compare && !failed {
		var shadow *sceneanalysis.ShadowEstimate
		if model != nil {
			estimate, err := sceneanalysis.ShadowMargin(rows[0].Assessment.Requirements, rows[1].Assessment.Requirements, *model)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(2)
			}
			shadow = &estimate
		}
		value = struct {
			Scenes     []result                      `json:"scenes"`
			Comparison sceneanalysis.Comparison      `json:"comparison"`
			Shadow     *sceneanalysis.ShadowEstimate `json:"shadow,omitempty"`
		}{rows, sceneanalysis.Compare(rows[0].Assessment.Requirements, rows[1].Assessment.Requirements, nil), shadow}
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if failed {
		os.Exit(1)
	}
}
