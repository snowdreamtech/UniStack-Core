package main

import (
	"bufio"
	"compress/gzip"
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

// ServiceMapping maps a project to its native OS service name.
type ServiceMapping struct {
	Project       string
	DebianService string
	RhelService   string
	AlpineService string
}

// FetchServicesData downloads OS package content indices to find `.service` and `init.d` files.
func FetchServicesData(ctx context.Context, pkgMappings map[string]*PackageMapping) (map[string]*ServiceMapping, error) {
	log.Println("Starting Service mapping extraction...")
	
	services := make(map[string]*ServiceMapping)

	// Pre-fill the services map for every known project
	for proj := range pkgMappings {
		services[proj] = &ServiceMapping{Project: proj}
	}

	// 1. Debian/Ubuntu Services (parsing Contents-amd64.gz)
	// Example URL: http://ftp.debian.org/debian/dists/bookworm/main/Contents-amd64.gz
	err := fetchDebianContents(ctx, "http://ftp.debian.org/debian/dists/bookworm/main/Contents-amd64.gz", pkgMappings, services)
	if err != nil {
		log.Printf("Warning: Failed to fetch Debian services: %v", err)
	}

	// TODO: Add RHEL (e.g. from repo filelists.xml.gz) and Alpine (APKINDEX) fetchers
	log.Println("RHEL and Alpine service scanning is currently a placeholder for future extension.")

	return services, nil
}

func fetchDebianContents(ctx context.Context, url string, pkgMappings map[string]*PackageMapping, services map[string]*ServiceMapping) error {
	log.Printf("Downloading Debian Contents index: %s", url)
	
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	
	var resp *http.Response
	var err error
	for attempt := 1; attempt <= MaxRetries; attempt++ {
		resp, err = http.DefaultClient.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			break
		}
		if resp != nil {
			resp.Body.Close()
		}
		if attempt == MaxRetries {
			return fmt.Errorf("failed after %d attempts: %w", MaxRetries, err)
		}
		time.Sleep(time.Duration(1<<attempt) * time.Second)
	}
	defer resp.Body.Close()

	gzReader, err := gzip.NewReader(resp.Body)
	if err != nil {
		return err
	}
	defer gzReader.Close()

	scanner := bufio.NewScanner(gzReader)
	for scanner.Scan() {
		line := scanner.Text()
		
		// We are looking for lines ending with .service 
		// Format: lib/systemd/system/nginx.service    web/nginx
		if strings.Contains(line, ".service") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				filepath := parts[0]
				pkgStr := parts[1] // could be multiple comma-separated packages

				if strings.HasSuffix(filepath, ".service") && (strings.HasPrefix(filepath, "lib/systemd/system/") || strings.HasPrefix(filepath, "usr/lib/systemd/system/")) {
					serviceName := filepath[strings.LastIndex(filepath, "/")+1 : len(filepath)-8]
					
					pkgs := strings.Split(pkgStr, ",")
					for _, pkgPath := range pkgs {
						// Extract actual package name (e.g., web/nginx -> nginx)
						pkgParts := strings.Split(pkgPath, "/")
						pkgName := pkgParts[len(pkgParts)-1]

						// Find which project this Debian package belongs to
						for proj, mapping := range pkgMappings {
							if mapping.DebianPkg == pkgName {
								if services[proj] != nil && services[proj].DebianService == "" {
									services[proj].DebianService = serviceName
								}
							}
						}
					}
				}
			}
		}
	}
	
	return scanner.Err()
}
