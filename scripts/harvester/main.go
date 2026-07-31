package main

import (
	"context"
	"log"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// HarvesterOutput defines the schema for services_dict.yml
type HarvesterOutput struct {
	Projects map[string]ProjectData `yaml:"projects"`
}

type ProjectData struct {
	Packages OSData `yaml:"packages,omitempty"`
	Services OSData `yaml:"services,omitempty"`
}

type OSData struct {
	Debian string `yaml:"debian,omitempty"`
	Rhel   string `yaml:"rhel,omitempty"`
	Alpine string `yaml:"alpine,omitempty"`
}

func main() {
	log.Println("=== UniStack Harvester Started ===")
	
	// Set a very generous timeout of 300 minutes (5 hours) for massive generations
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Minute)
	defer cancel()

	// 1. Fetch Repology Packages mapping
	pkgMappings, err := FetchRepologyData(ctx)
	if err != nil {
		log.Fatalf("Failed to fetch repology data: %v", err)
	}

	// 2. Fetch OS Services mapping
	srvMappings, err := FetchServicesData(ctx, pkgMappings)
	if err != nil {
		log.Fatalf("Failed to fetch services data: %v", err)
	}

	// 3. Combine into final dict
	output := HarvesterOutput{
		Projects: make(map[string]ProjectData),
	}

	for proj, pkg := range pkgMappings {
		srv := srvMappings[proj]
		
		projData := ProjectData{
			Packages: OSData{
				Debian: pkg.DebianPkg,
				Rhel:   pkg.RhelPkg,
				Alpine: pkg.AlpinePkg,
			},
		}

		if srv != nil {
			projData.Services = OSData{
				Debian: srv.DebianService,
				Rhel:   srv.RhelService,
				Alpine: srv.AlpineService,
			}
		}

		output.Projects[proj] = projData
	}

	// 4. Write services_dict.yml
	file, err := os.Create("services_dict.yml")
	if err != nil {
		log.Fatalf("Failed to create output file: %v", err)
	}
	defer file.Close()

	encoder := yaml.NewEncoder(file)
	encoder.SetIndent(2)
	if err := encoder.Encode(output); err != nil {
		log.Fatalf("Failed to write yaml: %v", err)
	}

	log.Println("Successfully generated services_dict.yml!")

	// 5. Generate Packages
	// Path assumes it is run from scripts/harvester
	if err := GeneratePackages("services_dict.yml", "../../packages", "templates"); err != nil {
		log.Fatalf("Package generation failed: %v", err)
	}
}
