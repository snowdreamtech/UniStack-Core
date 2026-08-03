---
apiVersion: "v1alpha1"
kind: "package"

metadata:
  name: "{{.Name}}"
  version: "{{.Version}}"
  description: "Auto-generated UniStack core package for {{.Name}}"
  authors: ["snowdreamtech <snowdreamtech@qq.com>"]
  homepage: "https://repology.org/project/{{.Name}}"
  license: "Unknown"
  tags: ["core", "auto-generated", "service"]

compatibility:
  - os: "linux"
    arch: ["amd64", "arm64", "386", "arm"]

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
