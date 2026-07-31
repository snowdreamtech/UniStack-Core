package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/klauspost/compress/zstd"
)

const (
	RepologyDumpURL = "https://dumps.repology.org/repology-database-dump-latest.sql.zst"
	MaxRetries      = 3
)

// PackageMapping holds the discovered native OS package names for a given project.
type PackageMapping struct {
	Project   string
	DebianPkg string
	RhelPkg   string
	AlpinePkg string
}

// FetchRepologyData downloads and parses the Repology ZST dump stream to extract package mappings.
func FetchRepologyData(ctx context.Context) (map[string]*PackageMapping, error) {
	log.Println("Starting Repology dump fetch with stream parsing...")

	var resp *http.Response
	var err error

	// Exponential backoff retry mechanism
	for attempt := 1; attempt <= MaxRetries; attempt++ {
		req, _ := http.NewRequestWithContext(ctx, "GET", RepologyDumpURL, nil)
		resp, err = http.DefaultClient.Do(req)
		
		if err == nil && resp.StatusCode == http.StatusOK {
			break
		}
		
		if resp != nil {
			resp.Body.Close()
		}
		
		log.Printf("Attempt %d failed: %v. Retrying...", attempt, err)
		if attempt == MaxRetries {
			return nil, fmt.Errorf("failed to fetch repology dump after %d attempts: %w", MaxRetries, err)
		}
		time.Sleep(time.Duration(1<<attempt) * time.Second)
	}
	defer resp.Body.Close()

	// Setup Zstandard decompressor
	decoder, err := zstd.NewReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to create zstd reader: %w", err)
	}
	defer decoder.Close()

	return parseRepologySQLStream(decoder)
}

func parseRepologySQLStream(reader io.Reader) (map[string]*PackageMapping, error) {
	scanner := bufio.NewScanner(reader)
	// Some lines might be very long in SQL dumps, allocate a larger buffer
	const maxCapacity = 10 * 1024 * 1024
	buf := make([]byte, maxCapacity)
	scanner.Buffer(buf, maxCapacity)

	mappings := make(map[string]*PackageMapping)
	inPackagesTable := false

	log.Println("Scanning decompressed stream...")

	var linesProcessed int64
	for scanner.Scan() {
		linesProcessed++
		if linesProcessed%10000000 == 0 {
			log.Printf("Processed %d lines...", linesProcessed)
		}

		line := scanner.Text()

		if inPackagesTable {
			if line == "\\." {
				inPackagesTable = false
				log.Printf("Finished reading packages table (total projects mapped: %d).", len(mappings))
				break // We only need the packages table
			}

			// Parse the TSV data for packages table
			cols := strings.Split(line, "\t")
			if len(cols) < 4 {
				continue
			}

			repo := cols[0]
			// subrepo := cols[1]
			project := cols[2]
			visiblename := cols[3]

			// Only process if it's one of the target OS families
			isDebian := strings.HasPrefix(repo, "debian_")
			isAlpine := strings.HasPrefix(repo, "alpine_")
			isRhel := strings.HasPrefix(repo, "epel_") || strings.HasPrefix(repo, "centos_") || strings.HasPrefix(repo, "rocky_") || strings.HasPrefix(repo, "rhel_") || strings.HasPrefix(repo, "fedora_")

			if isDebian || isAlpine || isRhel {
				mapping, exists := mappings[project]
				if !exists {
					mapping = &PackageMapping{Project: project}
					mappings[project] = mapping
				}

				// Map to specific OS fields
				if isDebian && mapping.DebianPkg == "" {
					mapping.DebianPkg = visiblename
				}
				if isAlpine && mapping.AlpinePkg == "" {
					mapping.AlpinePkg = visiblename
				}
				if isRhel && mapping.RhelPkg == "" {
					mapping.RhelPkg = visiblename
				}
			}
		} else {
			// Check if we reached the packages COPY statement
			// The exact statement might be "COPY public.packages (" or "COPY packages ("
			if strings.HasPrefix(line, "COPY packages ") || strings.HasPrefix(line, "COPY public.packages ") {
				log.Println("Found packages table. Starting extraction...")
				inPackagesTable = true
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading stream: %w", err)
	}

	return mappings, nil
}
