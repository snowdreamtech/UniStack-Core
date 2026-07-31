---
apiVersion: "v1alpha1"
kind: "package"

metadata:
  name: "{{.Name}}"
  version: "{{.Version}}"
  description: "Auto-generated UniStack core package for {{.Name}}"
  authors: ["UniStack Harvester <harvester@unistack.org>"]
  homepage: "https://repology.org/project/{{.Name}}"
  license: "Unknown"
  tags: ["core", "auto-generated", "service"]

compatibility:
  - os: "linux"
    arch: ["amd64", "arm64", "386", "arm"]
  - os: "darwin"
    arch: ["amd64", "arm64"]

delivery:
  type: "app"
  
dependencies:
  required: {}
  recommended: {}

spec:
  packages:
    debian: "{{.DebianPkg}}"
    rhel: "{{.RhelPkg}}"
    alpine: "{{.AlpinePkg}}"
  services:
    debian: "{{.DebianService}}"
    rhel: "{{.RhelService}}"
    alpine: "{{.AlpineService}}"
