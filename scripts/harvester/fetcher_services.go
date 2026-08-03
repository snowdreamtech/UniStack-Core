package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
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

	// Build reverse index: rhelPkgName -> projectName
	rhelPkgToProject := make(map[string]string, len(pkgMappings))
	for proj, mapping := range pkgMappings {
		if mapping.RhelPkg != "" {
			rhelPkgToProject[mapping.RhelPkg] = proj
		}
	}

	// 2. RHEL Services (parsing Rocky Linux 9 BaseOS and AppStream repodata)
	rhelRepos := []string{
		"http://dl.rockylinux.org/pub/rocky/9/BaseOS/x86_64/os/",
		"http://dl.rockylinux.org/pub/rocky/9/AppStream/x86_64/os/",
	}
	for _, repoUrl := range rhelRepos {
		if err := fetchRhelContents(ctx, repoUrl, rhelPkgToProject, services); err != nil {
			log.Printf("Warning: Failed to fetch RHEL services from %s: %v", repoUrl, err)
		}
	}

	// 3. Alpine Heuristic Mapping (Fallback to Debian/RHEL for OpenRC)
	// Alpine APKINDEX doesn't contain file lists natively. We use a high-confidence heuristic:
	// OpenRC init scripts are usually identically named to the package name or the Systemd service.
	for _, srv := range services {
		if srv.AlpineService == "" {
			if srv.DebianService != "" {
				srv.AlpineService = srv.DebianService
			} else if srv.RhelService != "" {
				srv.AlpineService = srv.RhelService
			}
		}
	}
	log.Println("Applied heuristic mapping for Alpine services.")

	return services, nil
}

func fetchRhelContents(ctx context.Context, repoUrl string, rhelPkgToProject map[string]string, services map[string]*ServiceMapping) error {
	log.Printf("Fetching RHEL repomd from %s...", repoUrl)
	req, err := http.NewRequestWithContext(ctx, "GET", repoUrl+"repodata/repomd.xml", nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("bad status fetching repomd.xml: %d", resp.StatusCode)
	}

	// 1. Read repomd.xml completely into a string buffer to extract the filelists location
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	bodyStr := string(bodyBytes)
	
	// Find <data type="filelists">...<location href="..."/>
	filelistsHref := ""
	dataParts := strings.Split(bodyStr, `<data type="filelists">`)
	if len(dataParts) > 1 {
		locParts := strings.Split(dataParts[1], `<location href="`)
		if len(locParts) > 1 {
			filelistsHref = strings.Split(locParts[1], `"`)[0]
		}
	}

	if filelistsHref == "" {
		return fmt.Errorf("could not find filelists location in repomd.xml")
	}

	// 2. Fetch the actual filelists.xml.gz
	filelistsUrl := repoUrl + filelistsHref
	log.Printf("Downloading RHEL filelists from %s...", filelistsUrl)
	req2, err := http.NewRequestWithContext(ctx, "GET", filelistsUrl, nil)
	if err != nil {
		return err
	}

	resp2, err := http.DefaultClient.Do(req2)
	if err != nil {
		return err
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != 200 {
		return fmt.Errorf("bad status fetching filelists.xml.gz: %d", resp2.StatusCode)
	}

	gzReader, err := gzip.NewReader(resp2.Body)
	if err != nil {
		return err
	}
	defer gzReader.Close()

	// 3. Stream parse the XML to find <package name="xxx">...<file>/usr/lib/systemd/system/yyy.service</file>
	// We use manual byte scanning similar to Debian for ultimate performance, as encoding/xml is heavy.
	scanner := bufio.NewScanner(gzReader)
	
	// RHEL filelists.xml can have very long lines if not prettified, but it's typically line broken per <file>
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 10*1024*1024)

	var currentPkg string
	
	for scanner.Scan() {
		line := scanner.Bytes()
		
		// If line defines a package
		if bytes.Contains(line, []byte("<package ")) && bytes.Contains(line, []byte(`name="`)) {
			// Extract name="..."
			strLine := string(line)
			idx := strings.Index(strLine, `name="`)
			if idx != -1 {
				endIdx := strings.Index(strLine[idx+6:], `"`)
				if endIdx != -1 {
					currentPkg = strLine[idx+6 : idx+6+endIdx]
				}
			}
			continue
		}

		// If line defines a file
		if bytes.Contains(line, []byte("<file>")) && bytes.Contains(line, []byte(".service</file>")) {
			if !bytes.Contains(line, []byte("/usr/lib/systemd/system/")) && !bytes.Contains(line, []byte("/lib/systemd/system/")) {
				continue
			}
			if currentPkg == "" {
				continue
			}

			strLine := string(line)
			start := strings.Index(strLine, "<file>") + 6
			end := strings.Index(strLine, "</file>")
			if start >= 6 && end > start {
				filePath := strLine[start:end]
				serviceName := filePath[strings.LastIndex(filePath, "/")+1 : len(filePath)-8] // remove .service

				// Fuzzy matching logic identical to Debian
				partsDash := strings.Split(currentPkg, "-")
				matched := false
				for i := len(partsDash); i > 0; i-- {
					subPkg := strings.Join(partsDash[:i], "-")
					if proj, ok := rhelPkgToProject[subPkg]; ok {
						if services[proj] != nil && services[proj].RhelService == "" {
							services[proj].RhelService = serviceName
						}
						matched = true
						break
					}
				}
				
				if !matched {
					if proj, ok := rhelPkgToProject[serviceName]; ok {
						if services[proj] != nil && services[proj].RhelService == "" {
							services[proj].RhelService = serviceName
						}
					}
				}
			}
		}
	}

	return scanner.Err()
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

			// O(1) reverse index lookup with fallback for sub-packages
			// Debian often splits projects: e.g. project 'mariadb' provides service via 'mariadb-server-10.5'
			partsDash := strings.Split(pkgName, "-")
			matched := false
			for i := len(partsDash); i > 0; i-- {
				subPkg := strings.Join(partsDash[:i], "-")
				if proj, ok := debianPkgToProject[subPkg]; ok {
					if services[proj] != nil && services[proj].DebianService == "" {
						services[proj].DebianService = serviceName
					}
					matched = true
					break
				}
			}
			
			if !matched {
				// Fallback: check if the service name itself matches a project
				// E.g., if package is "nginx-core" but project is "nginx" and service is "nginx"
				if proj, ok := debianPkgToProject[serviceName]; ok {
					if services[proj] != nil && services[proj].DebianService == "" {
						services[proj].DebianService = serviceName
					}
				}
			}
		}
	}
	
	return scanner.Err()
}
