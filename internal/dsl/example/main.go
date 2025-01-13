package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/get-glu/glu/internal/dsl/parser"
)

func main() {
	// Read the pipeline.glu file in the current directory
	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	content, err := os.ReadFile(filepath.Join(dir, "pipeline.glu"))
	if err != nil {
		panic(err)
	}

	// Parse the pipeline
	systems, err := parser.ParsePipeline(string(content))
	if err != nil {
		panic(err)
	}

	// Example: Print out the parsed structure
	for systemName, pipelines := range systems {
		fmt.Printf("System: %s\n", systemName)
		for _, pipeline := range pipelines {
			fmt.Printf("  Pipeline: %s\n", pipeline.Name)

			fmt.Println("  Sources:")
			for sourceName, source := range pipeline.Sources {
				fmt.Printf("    %s: type=%s, name=%s\n",
					sourceName, source.Type, source.Name)
			}

			fmt.Println("  Phases:")
			for phaseName, phase := range pipeline.Phases {
				fmt.Printf("    %s: source=%s, promotes_from=%s\n",
					phaseName, phase.SourceRef, phase.PromotesFrom)
				if len(phase.Labels) > 0 {
					fmt.Println("    Labels:")
					for k, v := range phase.Labels {
						fmt.Printf("      %s: %s\n", k, v)
					}
				}
			}
		}
	}
}
