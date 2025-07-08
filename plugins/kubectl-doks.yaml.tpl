apiVersion: krew.googlecontainertools.github.com/v1alpha2
kind: Plugin
metadata:
  name: doks
spec:
  version: v__VERSION__
  homepage: https://github.com/DO-Solutions/kubectl-doks
  shortDescription: "A kubectl plugin for interacting with DigitalOcean Kubernetes clusters."
  description: |
    The DigitalOcean kubectl plugin allows you to interact with your DOKS clusters.
    It can be used to:
    - Add a DOKS cluster to your kubeconfig
    - Get the kubeconfig for a DOKS cluster
    - Get credentials for a DOKS cluster
  platforms:
    - selector: {matchLabels: {os: darwin, arch: amd64}}
      uri: https://github.com/DO-Solutions/kubectl-doks/releases/download/v__VERSION__/kubectl-doks-darwin-amd64.tar.gz
      sha256: __DARWIN_AMD64_SHA256__
      files:
        - from: "*"
          to: "."
      bin: "kubectl-doks"
    - selector: {matchLabels: {os: darwin, arch: arm64}}
      uri: https://github.com/DO-Solutions/kubectl-doks/releases/download/v__VERSION__/kubectl-doks-darwin-arm64.tar.gz
      sha256: __DARWIN_ARM64_SHA256__
      files:
        - from: "*"
          to: "."
      bin: "kubectl-doks"
    - selector: {matchLabels: {os: linux, arch: amd64}}
      uri: https://github.com/DO-Solutions/kubectl-doks/releases/download/v__VERSION__/kubectl-doks-linux-amd64.tar.gz
      sha256: __LINUX_AMD64_SHA256__
      files:
        - from: "*"
          to: "."
      bin: "kubectl-doks"
