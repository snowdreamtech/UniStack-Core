package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/klauspost/compress/zstd"
)

// validProjectName matches names containing only alphanumeric, hyphens, underscores, dots, and plus signs.
var validProjectName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._+\-]*$`)

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

	// Dynamic column indices for packages table
	var repoIdx, projectIdx, visiblenameIdx = -1, -1, -1

	var linesProcessed int64
	for scanner.Scan() {
		linesProcessed++
		if linesProcessed%10000000 == 0 {
			log.Printf("Processed %d lines...", linesProcessed)
		}
		
		lineBytes := scanner.Bytes()
		var line string

		if inPackagesTable {
			line = string(lineBytes)
			if line == "\\." {
				inPackagesTable = false
				log.Printf("Finished reading packages table (total projects mapped: %d).\n", len(mappings))
				break
			}

			// Format: TSV
			cols := strings.Split(line, "\t")

			// Ensure we have enough columns based on our dynamic indices
			maxIdx := repoIdx
			if projectIdx > maxIdx { maxIdx = projectIdx }
			if visiblenameIdx > maxIdx { maxIdx = visiblenameIdx }
			
			if len(cols) <= maxIdx {
				continue
			}

			repo := cols[repoIdx]
			project := cols[projectIdx]
			visiblename := cols[visiblenameIdx]

			isDebian := strings.HasPrefix(repo, "debian_")
			isAlpine := strings.HasPrefix(repo, "alpine_")
			isRhel := strings.HasPrefix(repo, "epel_") || strings.HasPrefix(repo, "centos_") || strings.HasPrefix(repo, "rocky_") || strings.HasPrefix(repo, "rhel_") || strings.HasPrefix(repo, "fedora_")

			if isDebian || isAlpine || isRhel {
				if !validProjectName.MatchString(project) {
					continue
				}
				mapping, exists := mappings[project]
				if !exists {
					mapping = &PackageMapping{
						Project: project,
					}
					mappings[project] = mapping
				}

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
			// Quick byte check before string allocation
			if bytes.HasPrefix(lineBytes, []byte("COPY ")) {
				line = string(lineBytes)
				// Check if we reached the packages COPY statement
				if strings.HasPrefix(line, "COPY packages ") || 
			   strings.HasPrefix(line, "COPY public.packages ") || 
			   strings.HasPrefix(line, "COPY repology.packages ") || 
			   strings.HasPrefix(line, "COPY \"packages\" ") || 
			   strings.HasPrefix(line, "COPY public.\"packages\" ") || 
			   strings.HasPrefix(line, "COPY repology.\"packages\" ") ||
			   strings.HasPrefix(line, "COPY packages(") ||
			   strings.HasPrefix(line, "COPY public.packages(") ||
			   strings.HasPrefix(line, "COPY repology.packages(") {
			inPackagesTable = true
			
			// Parse the column names from the COPY header to adapt to schema changes
			start := strings.Index(line, "(")
			end := strings.Index(line, ")")
			if start != -1 && end != -1 {
				headerStr := line[start+1 : end]
				headers := strings.Split(headerStr, ",")
				for i, h := range headers {
					h = strings.TrimSpace(h)
					// Remove any quotes
					h = strings.ReplaceAll(h, "\"", "")
					if h == "repo" { repoIdx = i }
					if h == "project" || h == "effname" { projectIdx = i }
					if h == "visiblename" { visiblenameIdx = i }
				}
			}
			
			// Fallbacks in case columns weren't explicitly listed in the COPY statement
			if repoIdx == -1 { repoIdx = 1 } 
			if projectIdx == -1 { projectIdx = 27 } // effname is 27 in newer schema, project was 2 in old schema
			if visiblenameIdx == -1 { visiblenameIdx = 9 }
			
			log.Printf("Found packages table. Extraction initialized with cols: repo=%d, project=%d, visiblename=%d\n", repoIdx, projectIdx, visiblenameIdx)
		}
		}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading stream: %w", err)
	}

	return mappings, nil
}
