// Demo program showing indexdir functionality
package main

import (
	"fmt"
	"os"
	
	"github.com/dundee/gdu/v5/pkg/indexdir"
)

func main() {
	fmt.Println("=== gdu indexdir Demo ===\n")
	
	// 1. Config directory
	cfg, err := indexdir.ConfigDir()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("✓ Config directory: %s\n", cfg)
	
	// 2. Index directory
	idx, err := indexdir.IndexDir()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("✓ Index directory: %s\n", idx)
	
	// 3. Path encoding
	testPaths := []string{
		"/home/alice/projects",
		"/",
		"/usr/local/bin",
	}
	
	fmt.Println("\n--- Path Encoding ---")
	for _, p := range testPaths {
		encoded := indexdir.EncodePath(p)
		fmt.Printf("  %s → %s.gds\n", p, encoded)
	}
	
	// 4. Index path resolution
	fmt.Println("\n--- Index Path Resolution ---")
	scanPath := "/home/alice/projects/myapp"
	idxPath, _ := indexdir.IndexPathForScan(scanPath)
	fmt.Printf("  Path: %s\n", scanPath)
	fmt.Printf("  Index: %s\n", idxPath)
	
	// 5. Ancestor resolution (simulation)
	fmt.Println("\n--- Ancestor-Aware Resolution ---")
	fmt.Println("  Scenario: /home/alice/projects already indexed")
	fmt.Println("  Request: gdu /home/alice/projects/myapp")
	
	// Simulate creating parent index
	parentPath := "/home/alice/projects"
	parentIdx, _ := indexdir.IndexPathForScan(parentPath)
	os.MkdirAll(idx, 0755)
	os.WriteFile(parentIdx, []byte("test"), 0644)
	
	// Now resolve subfolder
	resolvedIdx, resolvedRoot, _ := indexdir.ResolveIndexAndScanRoot("/home/alice/projects/myapp")
	fmt.Printf("  → Found parent index: %s\n", resolvedIdx)
	fmt.Printf("  → Will scan from: %s (parent, not subfolder!)\n", resolvedRoot)
	
	// Cleanup
	os.Remove(parentIdx)
	
	fmt.Println("\n✓ All demonstrations complete!")
	fmt.Println("\nSee EXAMPLES.md for usage examples")
	fmt.Println("See INTEGRATION.md for integration guide")
}
