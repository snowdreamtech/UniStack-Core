package main

import (
	"bufio"
	"bytes"
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

	// Build reverse index: debianPkgName -> projectName for O(1) lookup
	debianPkgToProject := make(map[string]string, len(pkgMappings))
	for proj, mapping := range pkgMappings {
		if mapping.DebianPkg != "" {
			debianPkgToProject[mapping.DebianPkg] = proj
		}
	}

	// 1. Debian/Ubuntu Services (parsing Contents-amd64.gz)
	// Example URL: http://ftp.debian.org/debian/dists/bookworm/main/Contents-amd64.gz
	err := fetchDebianContents(ctx, "http://ftp.debian.org/debian/dists/bookworm/main/Contents-amd64.gz", debianPkgToProject, services)
	if err != nil {
		log.Printf("Warning: Failed to fetch Debian services: %v", err)
	}

	// TODO: Add RHEL (e.g. from repo filelists.xml.gz) and Alpine (APKINDEX) fetchers
	log.Println("RHEL and Alpine service scanning is currently a placeholder for future extension.")

	return services, nil
}

func fetchDebianContents(ctx context.Context, url string, debianPkgToProject map[string]string, services map[string]*ServiceMapping) error {
	log.Printf("Downloading Debian Contents index: %s", url)
	
	var resp *http.Response
	var err error
	for attempt := 1; attempt <= MaxRetries; attempt++ {
		req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
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
		lineBytes := scanner.Bytes()
		
		// Quick heuristic filter: skip allocations for the 99.9% of lines that are not systemd services
		if !bytes.Contains(lineBytes, []byte(".service")) {
			continue
		}

		line := string(lineBytes)
		// Format: lib/systemd/system/nginx.service    web/nginx
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		filePath := parts[0]
		if !strings.HasSuffix(filePath, ".service") {
			continue
		}
		if !strings.HasPrefix(filePath, "lib/systemd/system/") && !strings.HasPrefix(filePath, "usr/lib/systemd/system/") {
			continue
		}

		serviceName := filePath[strings.LastIndex(filePath, "/")+1 : len(filePath)-8]
		pkgStr := parts[1]

		for _, pkgPath := range strings.Split(pkgStr, ",") {
			// Extract actual package name (e.g., web/nginx -> nginx)
			pkgParts := strings.Split(pkgPath, "/")
			pkgName := pkgParts[len(pkgParts)-1]

			// O(1) reverse index lookup instead of O(N) full scan
			if proj, ok := debianPkgToProject[pkgName]; ok {
				if services[proj] != nil && services[proj].DebianService == "" {
					services[proj].DebianService = serviceName
				}
			}
		}
	}
	
	return scanner.Err()
}
