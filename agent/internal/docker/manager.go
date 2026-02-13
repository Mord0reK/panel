package docker

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/moby/moby/client"
)

type ContainerManager struct {
	cli               *client.Client
	remoteDigestCache map[string]cachedDigestResult
	cacheMu           sync.RWMutex
}

func NewContainerManager(cli *client.Client) *ContainerManager {
	return &ContainerManager{
		cli:               cli,
		remoteDigestCache: make(map[string]cachedDigestResult),
	}
}

type cachedDigestResult struct {
	digest string
	errMsg string
}

func (m *ContainerManager) StopContainer(ctx context.Context, containerID string) error {
	cmd := exec.Command("docker", "stop", containerID)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to stop container %s: %w\n%s", containerID, err, output)
	}
	return nil
}

func (m *ContainerManager) StartContainer(ctx context.Context, containerID string) error {
	cmd := exec.Command("docker", "start", containerID)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to start container %s: %w\n%s", containerID, err, output)
	}
	return nil
}

func (m *ContainerManager) RestartContainer(ctx context.Context, containerID string) error {
	cmd := exec.Command("docker", "restart", containerID)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to restart container %s: %w\n%s", containerID, err, output)
	}
	return nil
}

type UpdateInfo struct {
	ContainerName   string `json:"container_name"`
	CurrentImage    string `json:"current_image"`
	LatestImage     string `json:"latest_image"`
	UpdateAvailable bool   `json:"update_available"`
	Status          string `json:"status,omitempty"`
	Error           string `json:"error,omitempty"`
	Project         string `json:"project,omitempty"`
	Service         string `json:"service,omitempty"`
}

const (
	UpdateStatusUpToDate  = "up_to_date"
	UpdateStatusAvailable = "update_available"
	UpdateStatusRateLimit = "rate_limited"
	UpdateStatusUnknown   = "unknown"
)

type UpdateResult struct {
	Container string `json:"container"`
	Success   bool   `json:"success"`
	Message   string `json:"message"`
}

func (m *ContainerManager) CheckForUpdates(ctx context.Context, containerID string) ([]UpdateInfo, error) {
	inspect, err := m.cli.ContainerInspect(ctx, containerID, client.ContainerInspectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to inspect container: %w", err)
	}
	containerName := strings.TrimPrefix(inspect.Container.Name, "/")

	if inspect.Container.Config == nil {
		return nil, fmt.Errorf("container config unavailable")
	}

	currentImage := strings.TrimSpace(inspect.Container.Config.Image)
	if currentImage == "" {
		return nil, fmt.Errorf("container image is empty")
	}
	imageName := normalizeImageRef(currentImage)

	info := UpdateInfo{
		ContainerName:   containerName,
		CurrentImage:    currentImage,
		LatestImage:     imageName,
		UpdateAvailable: false,
		Status:          UpdateStatusUnknown,
	}

	if inspect.Container.Config.Labels != nil {
		info.Project = strings.TrimSpace(inspect.Container.Config.Labels["com.docker.compose.project"])
		info.Service = strings.TrimSpace(inspect.Container.Config.Labels["com.docker.compose.service"])
	}

	localDigest, err := m.getLocalImageDigest(ctx, imageName)
	if err != nil {
		info.Status = UpdateStatusUnknown
		info.Error = err.Error()
		return []UpdateInfo{info}, nil
	}

	remoteDigest, err := m.getRemoteImageDigest(ctx, imageName)
	if err != nil {
		if isRateLimitError(err) {
			info.Status = UpdateStatusRateLimit
		} else {
			info.Status = UpdateStatusUnknown
		}
		info.Error = err.Error()
		return []UpdateInfo{info}, nil
	}

	if remoteDigest != localDigest {
		info.UpdateAvailable = true
		info.Status = UpdateStatusAvailable
	} else {
		info.Status = UpdateStatusUpToDate
	}

	return []UpdateInfo{info}, nil
}

func (m *ContainerManager) UpdateContainer(ctx context.Context, containerID string) ([]UpdateResult, error) {
	cmd := exec.Command("docker", "inspect", "--format", "{{.Name}}", containerID)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to inspect container: %w", err)
	}
	containerName := strings.TrimPrefix(strings.TrimSpace(string(out)), "/")

	cmd = exec.Command("docker", "inspect", "--format", "{{index .Config.Labels \"com.docker.compose.project\"}}", containerID)
	out, _ = cmd.Output()
	project := strings.TrimSpace(string(out))

	cmd = exec.Command("docker", "inspect", "--format", "{{index .Config.Labels \"com.docker.compose.project.working_dir\"}}", containerID)
	out, _ = cmd.Output()
	workingDir := strings.TrimSpace(string(out))

	if project != "" {
		cmd = exec.Command("docker", "compose", "-p", project, "pull")
		if workingDir != "" {
			cmd.Dir = workingDir
		}
		output, err := cmd.CombinedOutput()
		if err != nil {
			return []UpdateResult{{Container: containerName, Success: false, Message: fmt.Sprintf("failed to pull: %s", output)}}, nil
		}

		cmd = exec.Command("docker", "compose", "-p", project, "up", "-d", "--force-recreate")
		if workingDir != "" {
			cmd.Dir = workingDir
		}
		output, err = cmd.CombinedOutput()
		if err != nil {
			return []UpdateResult{{Container: containerName, Success: false, Message: fmt.Sprintf("failed to recreate: %s", output)}}, nil
		}

		return []UpdateResult{{Container: containerName, Success: true, Message: "Container updated via compose"}}, nil
	}

	cmd = exec.Command("docker", "inspect", "--format", "{{.Config.Image}}", containerID)
	out, err = cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get image: %w", err)
	}
	imageName := strings.TrimSpace(string(out))

	result := []UpdateResult{{Container: containerName}}

	cmd = exec.Command("docker", "pull", imageName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		result[0].Success = false
		result[0].Message = fmt.Sprintf("failed to pull image: %s", output)
		return result, nil
	}

	cmd = exec.Command("docker", "stop", containerID)
	_, err = cmd.Output()
	if err != nil {
		result[0].Success = false
		result[0].Message = fmt.Sprintf("failed to stop: %v", err)
		return result, nil
	}

	cmd = exec.Command("docker", "rm", containerID)
	_, err = cmd.Output()
	if err != nil {
		result[0].Success = false
		result[0].Message = fmt.Sprintf("failed to remove: %v", err)
		return result, nil
	}

	cmd = exec.Command("docker", "run", "-d", "--name", containerName, imageName)
	_, err = cmd.Output()
	if err != nil {
		result[0].Success = false
		result[0].Message = fmt.Sprintf("failed to recreate: %v", err)
		return result, nil
	}

	result[0].Success = true
	result[0].Message = "Container updated successfully"
	return result, nil
}

func (m *ContainerManager) CheckComposeUpdates(ctx context.Context, projectName, workingDir string) ([]UpdateInfo, error) {
	cmd := exec.CommandContext(ctx, "docker", "compose", "-p", projectName, "config", "--images")
	if workingDir != "" {
		cmd.Dir = workingDir
	}
	output, err := cmd.Output()
	if err != nil {
		return []UpdateInfo{{
			Project:         projectName,
			UpdateAvailable: false,
			Status:          UpdateStatusUnknown,
			Error:           fmt.Sprintf("failed to resolve compose images: %v", err),
		}}, nil
	}

	images := strings.Split(strings.TrimSpace(string(output)), "\n")
	hasUpdate := false
	hasRateLimit := false
	unknownErrors := make([]string, 0)
	rateLimitErrors := make([]string, 0)

	for _, image := range images {
		image = normalizeImageRef(image)
		if image == "" {
			continue
		}

		localDigest, localErr := m.getLocalImageDigest(ctx, image)
		if localErr != nil {
			unknownErrors = append(unknownErrors, fmt.Sprintf("%s: %v", image, localErr))
			continue
		}

		remoteDigest, remoteErr := m.getRemoteImageDigest(ctx, image)
		if remoteErr != nil {
			if isRateLimitError(remoteErr) {
				hasRateLimit = true
				rateLimitErrors = append(rateLimitErrors, fmt.Sprintf("%s: %v", image, remoteErr))
			} else {
				unknownErrors = append(unknownErrors, fmt.Sprintf("%s: %v", image, remoteErr))
			}
			continue
		}

		if remoteDigest != "" && remoteDigest != localDigest {
			hasUpdate = true
		}
	}

	info := UpdateInfo{
		Project:         projectName,
		UpdateAvailable: hasUpdate,
		Status:          UpdateStatusUpToDate,
	}

	if hasUpdate {
		info.Status = UpdateStatusAvailable
	} else if hasRateLimit {
		info.Status = UpdateStatusRateLimit
		info.Error = strings.Join(rateLimitErrors, "; ")
	} else if len(unknownErrors) > 0 {
		info.Status = UpdateStatusUnknown
		info.Error = strings.Join(unknownErrors, "; ")
	}

	return []UpdateInfo{info}, nil
}

func (m *ContainerManager) getLocalImageDigest(ctx context.Context, image string) (string, error) {
	inspect, err := m.cli.ImageInspect(ctx, image)
	if err != nil {
		return "", fmt.Errorf("failed to inspect local image: %w", err)
	}

	if len(inspect.RepoDigests) == 0 {
		return "", fmt.Errorf("local digest is empty")
	}

	for _, repoDigest := range inspect.RepoDigests {
		digest, ok := digestFromRepoDigest(repoDigest)
		if !ok {
			continue
		}

		normalizedRepo := normalizeImageRef(strings.Split(repoDigest, "@")[0])
		if sameImageRepo(normalizedRepo, image) {
			return digest, nil
		}
	}

	for _, repoDigest := range inspect.RepoDigests {
		digest, ok := digestFromRepoDigest(repoDigest)
		if ok {
			return digest, nil
		}
	}

	return "", fmt.Errorf("local digest format invalid")
}

func (m *ContainerManager) getRemoteImageDigest(ctx context.Context, image string) (string, error) {
	m.cacheMu.RLock()
	if cached, ok := m.remoteDigestCache[image]; ok {
		m.cacheMu.RUnlock()
		if cached.errMsg != "" {
			return "", fmt.Errorf("%s", cached.errMsg)
		}
		return cached.digest, nil
	}
	m.cacheMu.RUnlock()

	inspectResult, err := m.cli.DistributionInspect(ctx, image, client.DistributionInspectOptions{})
	if err != nil {
		errMsg := fmt.Sprintf("failed to inspect remote digest: %v", err)
		m.cacheRemoteDigest(image, "", errMsg)
		return "", fmt.Errorf("%s", errMsg)
	}

	remoteDigest := strings.TrimSpace(inspectResult.Descriptor.Digest.String())
	if remoteDigest == "" {
		errMsg := "remote digest unavailable"
		m.cacheRemoteDigest(image, "", errMsg)
		return "", fmt.Errorf("%s", errMsg)
	}

	m.cacheRemoteDigest(image, remoteDigest, "")
	return remoteDigest, nil
}

func (m *ContainerManager) cacheRemoteDigest(image, digest, errMsg string) {
	m.cacheMu.Lock()
	m.remoteDigestCache[image] = cachedDigestResult{digest: digest, errMsg: errMsg}
	m.cacheMu.Unlock()
}

func (m *ContainerManager) UpdateComposeGroup(ctx context.Context, projectName, workingDir string) ([]UpdateResult, error) {
	cmd := exec.CommandContext(ctx, "docker", "compose", "-p", projectName, "pull")
	if workingDir != "" {
		cmd.Dir = workingDir
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to pull images: %w\n%s", err, output)
	}

	cmd = exec.CommandContext(ctx, "docker", "compose", "-p", projectName, "up", "-d")
	if workingDir != "" {
		cmd.Dir = workingDir
	}
	output, err = cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to recreate containers: %w\n%s", err, output)
	}

	return []UpdateResult{{
		Container: projectName,
		Success:   true,
		Message:   "Compose group updated successfully",
	}}, nil
}

func (m *ContainerManager) pullImage(ctx context.Context, imageName string) error {
	cmd := exec.CommandContext(ctx, "docker", "pull", imageName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to pull image: %w\n%s", err, output)
	}
	return nil
}

func normalizeImageRef(image string) string {
	image = strings.TrimSpace(image)
	if image == "" {
		return ""
	}
	// Handle sha256: prefix
	if strings.HasPrefix(image, "sha256:") {
		// Can't extract name from digest alone, return as-is
		// Docker will use local image if exists
		return image
	}
	// Handle @digest format
	parts := strings.Split(image, "@")
	if len(parts) > 0 {
		image = parts[0]
	}

	if !hasExplicitTag(image) {
		image += ":latest"
	}

	return image
}

func hasExplicitTag(image string) bool {
	lastSlash := strings.LastIndex(image, "/")
	lastColon := strings.LastIndex(image, ":")
	return lastColon > lastSlash
}

func digestFromRepoDigest(repoDigest string) (string, bool) {
	parts := strings.Split(strings.TrimSpace(repoDigest), "@")
	if len(parts) != 2 || strings.TrimSpace(parts[1]) == "" {
		return "", false
	}
	return strings.TrimSpace(parts[1]), true
}

func sameImageRepo(repo, image string) bool {
	repoWithoutTag := repo
	imageWithoutTag := image

	if idx := strings.LastIndex(repoWithoutTag, ":"); idx > strings.LastIndex(repoWithoutTag, "/") {
		repoWithoutTag = repoWithoutTag[:idx]
	}
	if idx := strings.LastIndex(imageWithoutTag, ":"); idx > strings.LastIndex(imageWithoutTag, "/") {
		imageWithoutTag = imageWithoutTag[:idx]
	}

	return repoWithoutTag == imageWithoutTag
}

func isRateLimitError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "toomanyrequests") ||
		strings.Contains(msg, "rate limit") ||
		strings.Contains(msg, "429")
}

func findComposeFile(projectName string) (string, error) {
	commonFiles := []string{
		"docker-compose.yml",
		"docker-compose.yaml",
		"compose.yml",
		"compose.yaml",
	}

	searchDirs := []string{".", "/opt/" + projectName, "/home/" + os.Getenv("USER") + "/" + projectName}

	for _, dir := range searchDirs {
		for _, file := range commonFiles {
			path := dir + "/" + file
			if _, err := os.Stat(path); err == nil {
				return path, nil
			}
		}
	}

	return "", fmt.Errorf("compose file not found for project %s", projectName)
}

func getComposeDir(composeFile string) string {
	parts := strings.Split(composeFile, "/")
	if len(parts) > 1 {
		return strings.Join(parts[:len(parts)-1], "/")
	}
	return "."
}

func FindContainerByName(ctx context.Context, cli *client.Client, name string) (string, error) {
	cmd := exec.CommandContext(ctx, "docker", "ps", "-a", "--format", "{{.ID}}")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to list containers: %w", err)
	}

	ids := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}

		cmd := exec.CommandContext(ctx, "docker", "inspect", "--format", "{{.Name}}", id)
		out, err := cmd.Output()
		if err != nil {
			continue
		}
		containerName := strings.TrimPrefix(strings.TrimSpace(string(out)), "/")

		cmd = exec.CommandContext(ctx, "docker", "inspect", "--format", "{{.Id}}", id)
		out, err = cmd.Output()
		if err != nil {
			continue
		}
		fullID := strings.TrimSpace(string(out))

		cleanName := strings.TrimPrefix(name, "/")
		shortID := fullID
		if len(fullID) > 12 {
			shortID = fullID[:12]
		}

		if containerName == cleanName || strings.HasPrefix(containerName, cleanName) || fullID == cleanName || shortID == cleanName {
			return fullID, nil
		}
	}

	return "", fmt.Errorf("container not found: %s", name)
}

func FindComposeProjectByContainer(ctx context.Context, cli *client.Client, containerID string) (string, error) {
	cmd := exec.CommandContext(ctx, "docker", "inspect", "--format", "{{index .Config.Labels \"com.docker.compose.project\"}}", containerID)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	project := strings.TrimSpace(string(out))
	if project != "" {
		return project, nil
	}
	return "", nil
}
