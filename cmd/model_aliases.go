// Package cmd - type aliases for the generated model names.
// This keeps command code readable while using go-swagger's generated types.
package cmd

import "github.com/flotio-dev/cli/pkg/api/models"

// Short aliases for the auto-generated model types.
type (
	LoginRequest        = models.GithubComFlotioDevCoreAPIInternalModulesUserModelLoginRequest
	AuthResponse        = models.GithubComFlotioDevCoreAPIInternalModulesUserModelAuthResponse
	RefreshTokenRequest = models.GithubComFlotioDevCoreAPIInternalModulesUserModelRefreshTokenRequest
	BuildRequest        = models.GithubComFlotioDevCoreAPIInternalModulesBuildModelBuildRequest
	BuildDTO            = models.GithubComFlotioDevCoreAPIInternalModulesBuildModelBuildDTO
	BuildResponse       = models.GithubComFlotioDevCoreAPIInternalModulesBuildModelBuildResponse
	BuildsResponse      = models.GithubComFlotioDevCoreAPIInternalModulesBuildModelBuildsResponse
	ProjectCreateReq    = models.InternalModulesProjectHandlerProjectCreateRequest
	ProjectUpdateReq    = models.InternalModulesProjectHandlerProjectUpdateRequest
	ProjectResp         = models.InternalModulesProjectHandlerProjectResponse
	ProjectsResp        = models.InternalModulesProjectHandlerProjectsResponse
	ProjectConfig       = models.GithubComFlotioDevCoreAPIInternalCommonDatabaseProjectConfig
	GHInstallResponse   = models.GithubComFlotioDevCoreAPIInternalModulesGithubModelGithubInstallationResponse
	GHPostInstallReq    = models.GithubComFlotioDevCoreAPIInternalModulesGithubModelPostInstallationRequest
)
