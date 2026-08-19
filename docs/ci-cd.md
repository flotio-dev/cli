# CI and releases

Pull requests and `main` run Go formatting, vet, race-enabled tests, coverage,
dependency review, Trivy, CodeQL and reproducible cross-compilation for Linux,
macOS and Windows. The stable required check is `ci-success`.

Only an explicit strict `vX.Y.Z` tag creates a GitHub release. The release uses
the already-built matrix artifacts, adds SHA-256 checksums and an SPDX SBOM,
creates GitHub-native provenance and SBOM attestations, verifies them, then
publishes the release. The workflow never creates tags automatically.

No repository secret is required. `GITHUB_TOKEN` receives `contents: write`
only in the release job, while OIDC and attestation writes are also confined to
that job. All other jobs are read-only.

Configure `main` protection with pull requests and required `ci-success`, a
ruleset restricting `v*` tag changes to release maintainers, Secret Scanning,
Push Protection, Dependabot security updates and the versioned CodeQL workflow.
Do not enable duplicate CodeQL default setup.

Local checks:

```bash
test -z "$(gofmt -l .)"
go vet ./...
go test -race ./...
go build ./...
actionlint .github/workflows/ci.yaml
```

To release, merge to `main`, wait for `ci-success`, create a signed or annotated
`vX.Y.Z` tag, and push that tag.
