package docker

import (
	"context"
	"fmt"
	"io"
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
	CurrentVersion  string `json:"current_version"`
	LatestImage     string `json:"latest_image"`
	LatestVersion   string `json:"latest_version"`
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

	// Normalizuj current image
	currentImageNormalized := normalizeImageRef(currentImage)

	// Dla sprawdzania update ZAWSZE porównuj z :latest
	latestImage := stripTag(currentImageNormalized) + ":latest"

	info := UpdateInfo{
		ContainerName:   containerName,
		CurrentImage:    currentImage,
		CurrentVersion:  "unknown",
		LatestImage:     latestImage,
		LatestVersion:   "unknown",
		UpdateAvailable: false,
		Status:          UpdateStatusUnknown,
	}

	if inspect.Container.Config.Labels != nil {
		info.Project = strings.TrimSpace(inspect.Container.Config.Labels["com.docker.compose.project"])
		info.Service = strings.TrimSpace(inspect.Container.Config.Labels["com.docker.compose.service"])
	}

	// Pobierz current version z local image
	currentImageInspect, err := m.cli.ImageInspect(ctx, currentImageNormalized)
	if err == nil {
		if version, ok := currentImageInspect.Config.Labels["org.opencontainers.image.version"]; ok && version != "" {
			info.CurrentVersion = version
		}
	}

	// Sprawdź lokalny digest
	localDigest, err := m.getLocalImageDigest(ctx, currentImageNormalized)
	if err != nil {
		info.Status = UpdateStatusUnknown
		info.Error = err.Error()
		return []UpdateInfo{info}, nil
	}

	// Sprawdź zdalny digest :latest
	remoteDigest, err := m.getRemoteImageDigest(ctx, latestImage)
	if err != nil {
		if isRateLimitError(err) {
			info.Status = UpdateStatusRateLimit
		} else {
			info.Status = UpdateStatusUnknown
		}
		info.Error = err.Error()
		return []UpdateInfo{info}, nil
	}

	// Jeśli digest się różni, pull :latest żeby zobaczyć wersję
	if remoteDigest != localDigest {
		info.UpdateAvailable = true
		info.Status = UpdateStatusAvailable

		// Pull latest w tle żeby wyciągnąć wersję
		pullResp, err := m.cli.ImagePull(ctx, latestImage, client.ImagePullOptions{})
		if err == nil {
			// Consume stream (musimy to zrobić żeby pull się skończył)
			_, _ = io.Copy(io.Discard, pullResp)
			pullResp.Close()

			// Teraz inspect pulled image
			latestImageInspect, err := m.cli.ImageInspect(ctx, latestImage)
			if err == nil {
				if version, ok := latestImageInspect.Config.Labels["org.opencontainers.image.version"]; ok && version != "" {
					info.LatestVersion = version
				}
			}
		}
	} else {
		info.Status = UpdateStatusUpToDate
		// Jak jest up to date, latest version = current version
		info.LatestVersion = info.CurrentVersion
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
	// ═══════════════════════════════════════════════════════════
	// ZAMIAST: docker compose config --images
	// UŻYWAMY: API do znalezienia kontenerów w projekcie
	// ═══════════════════════════════════════════════════════════
	filterArgs := client.Filters{}
	filterArgs = filterArgs.Add("label", fmt.Sprintf("com.docker.compose.project=%s", projectName))

	containers, err := m.cli.ContainerList(ctx, client.ContainerListOptions{
		All:     true,
		Filters: filterArgs,
	})
	if err != nil {
		return []UpdateInfo{{
			Project:         projectName,
			UpdateAvailable: false,
			Status:          UpdateStatusUnknown,
			Error:           fmt.Sprintf("failed to list containers: %v", err),
		}}, nil
	}

	if len(containers.Items) == 0 {
		return []UpdateInfo{{
			Project:         projectName,
			UpdateAvailable: false,
			Status:          UpdateStatusUnknown,
			Error:           "no containers found in project",
		}}, nil
	}

	// Zbierz unikalne obrazy z kontenerów
	imageSet := make(map[string]bool)
	for _, c := range containers.Items {
		image := normalizeImageRef(c.Image)
		if image != "" {
			imageSet[image] = true
		}
	}

	hasUpdate := false
	hasRateLimit := false
	unknownErrors := make([]string, 0)
	rateLimitErrors := make([]string, 0)

	// Sprawdź każdy unikalny obraz
	for image := range imageSet {
		latestImage := stripTag(image) + ":latest"

		localDigest, localErr := m.getLocalImageDigest(ctx, image)
		if localErr != nil {
			unknownErrors = append(unknownErrors, fmt.Sprintf("%s: %v", image, localErr))
			continue
		}

		remoteDigest, remoteErr := m.getRemoteImageDigest(ctx, latestImage)
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

// stripTag usuwa tag z image reference, zachowując repo
func stripTag(image string) string {
	image = strings.TrimSpace(image)
	if image == "" {
		return ""
	}

	// Usuń @digest jeśli jest
	if idx := strings.Index(image, "@"); idx != -1 {
		image = image[:idx]
	}

	// Usuń :tag jeśli jest
	lastSlash := strings.LastIndex(image, "/")
	lastColon := strings.LastIndex(image, ":")

	// Colon jest tagiem tylko jeśli jest po ostatnim slash
	if lastColon > lastSlash {
		image = image[:lastColon]
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
	containers, err := cli.ContainerList(ctx, client.ContainerListOptions{All: true})
	if err != nil {
		return "", fmt.Errorf("failed to list containers: %w", err)
	}

	cleanName := strings.TrimPrefix(name, "/")

	for _, c := range containers.Items {
		fullID := c.ID

		for _, containerName := range c.Names {
			containerName = strings.TrimPrefix(containerName, "/")

			shortID := fullID
			if len(fullID) > 12 {
				shortID = fullID[:12]
			}

			// TYLKO EXACT MATCH - usuń prefix match!
			if containerName == cleanName || fullID == cleanName || shortID == cleanName {
				return fullID, nil
			}
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
