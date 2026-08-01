package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

// GeneratePackages reads services_dict.yml and updates the packages/ directory
func GeneratePackages(dictPath string, packagesDir string, templatesDir string) error {
	log.Println("Starting Package Generation Phase...")

	// 1. Load services_dict.yml
	data, err := os.ReadFile(dictPath)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", dictPath, err)
	}

	var dict HarvesterOutput
	if err := yaml.Unmarshal(data, &dict); err != nil {
		return fmt.Errorf("failed to parse %s: %w", dictPath, err)
	}

	// 2. Load templates
	tmplOnly, err := template.ParseFiles(filepath.Join(templatesDir, "template_pkg_only.yml.tpl"))
	if err != nil {
		return fmt.Errorf("failed to load template_pkg_only.yml.tpl: %w", err)
	}

	tmplSvc, err := template.ParseFiles(filepath.Join(templatesDir, "template_pkg_with_service.yml.tpl"))
	if err != nil {
		return fmt.Errorf("failed to load template_pkg_with_service.yml.tpl: %w", err)
	}

	// 3. Iterate and Generate/Merge
	for proj, projData := range dict.Projects {
		// Implement directory exclusion rules (Task T014)
		if proj == "tasks" || proj == "templates" {
			continue
		}

		// Filter out projects that are likely not useful (e.g. ones with no debian/rhel mapping at all)
		if projData.Packages.Debian == "" && projData.Packages.Alpine == "" && projData.Packages.Rhel == "" {
			continue
		}

		targetDir := filepath.Join(packagesDir, proj)
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			log.Printf("Warning: failed to create directory %s: %v", targetDir, err)
			continue
		}

		targetFile := filepath.Join(targetDir, "package.yml")

		// Retrieve existing version to prevent overwriting with 0.0.0
		existingVersion := "0.0.0"
		if existingData, err := os.ReadFile(targetFile); err == nil {
			var existing struct {
				Metadata struct {
					Version string `yaml:"version"`
				} `yaml:"metadata"`
			}
			if err := yaml.Unmarshal(existingData, &existing); err == nil {
				if existing.Metadata.Version != "" && existing.Metadata.Version != "0.0.0" {
					existingVersion = existing.Metadata.Version
				}
			}
		}

		// Prepare template data
		tmplData := map[string]interface{}{
			"Name":          proj,
			"Version":       existingVersion, // Preserves manual versions or defaults to 0.0.0
			"DebianPkg":     projData.Packages.Debian,
			"RhelPkg":       projData.Packages.Rhel,
			"AlpinePkg":     projData.Packages.Alpine,
			"DebianService": projData.Services.Debian,
			"RhelService":   projData.Services.Rhel,
			"AlpineService": projData.Services.Alpine,
		}

		hasService := projData.Services.Debian != "" || projData.Services.Rhel != "" || projData.Services.Alpine != ""
		selectedTmpl := tmplOnly
		if hasService {
			selectedTmpl = tmplSvc
		}

		var buf bytes.Buffer
		if err := selectedTmpl.Execute(&buf, tmplData); err != nil {
			log.Printf("Warning: failed to execute template for %s: %v", proj, err)
			continue
		}

		// Perform AST Merge if file already exists
		if _, err := os.Stat(targetFile); err == nil {
			mergedContent, err := mergeYAML(targetFile, buf.Bytes())
			if err != nil {
				log.Printf("Warning: failed to merge YAML for %s: %v", proj, err)
				continue
			}
			if err := os.WriteFile(targetFile, mergedContent, 0644); err != nil {
				log.Printf("Warning: failed to write merged file %s: %v", targetFile, err)
			}
		} else {
			// Write new file
			if err := os.WriteFile(targetFile, buf.Bytes(), 0644); err != nil {
				log.Printf("Warning: failed to write new file %s: %v", targetFile, err)
			}
		}
	}

	log.Println("Package Generation Phase completed successfully!")
	return nil
}

// mergeYAML parses the existing file and the newly generated template,
// and intelligently merges the new auto-generated fields into the existing AST
// without destroying human-added nodes (like tasks/, custom scripts, etc.)
func mergeYAML(existingFile string, newContent []byte) ([]byte, error) {
	existingData, err := os.ReadFile(existingFile)
	if err != nil {
		return nil, err
	}

	var existingNode yaml.Node
	if err := yaml.Unmarshal(existingData, &existingNode); err != nil {
		return nil, err
	}

	var newNode yaml.Node
	if err := yaml.Unmarshal(newContent, &newNode); err != nil {
		return nil, err
	}

	// Deep merge logic (recursive AST merge)
	mergeNodes(&existingNode, &newNode)

	var out bytes.Buffer
	encoder := yaml.NewEncoder(&out)
	encoder.SetIndent(2)
	if err := encoder.Encode(&existingNode); err != nil {
		return nil, err
	}

	// Clean up formatting issues that sometimes happen with yaml.v3
	outStr := strings.ReplaceAll(out.String(), "    ", "  ")
	return []byte(outStr), nil
}

// mergeNodes performs a deep merge of yaml AST nodes.
// src overrides values in dst. Unrecognized fields in dst are kept intact.
func mergeNodes(dst, src *yaml.Node) {
	if dst.Kind != yaml.MappingNode || src.Kind != yaml.MappingNode {
		*dst = *src
		return
	}

	for i := 0; i < len(src.Content); i += 2 {
		srcKey := src.Content[i]
		srcVal := src.Content[i+1]

		found := false
		for j := 0; j < len(dst.Content); j += 2 {
			dstKey := dst.Content[j]
			dstVal := dst.Content[j+1]

			if dstKey.Value == srcKey.Value {
				// Key exists, merge recursively
				mergeNodes(dstVal, srcVal)
				found = true
				break
			}
		}

		if !found {
			// Append new key-value pair to dst
			dst.Content = append(dst.Content, srcKey, srcVal)
		}
	}
}
