package docker

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/moby/moby/client"
)

type ContainerManager struct {
	cli               *client.Client
	remoteDigestCache map[string]cachedDigestResult
	cacheMu           sync.RWMutex
}

const (
	cacheTTLSuccess   = 6 * time.Hour
	cacheTTLRateLimit = 15 * time.Minute
	cacheTTLError     = 1 * time.Hour
)

func NewContainerManager(cli *client.Client) *ContainerManager {
	return &ContainerManager{
		cli:               cli,
		remoteDigestCache: make(map[string]cachedDigestResult),
	}
}

type cachedDigestResult struct {
	digest    string
	errMsg    string
	expiresAt time.Time
}

func (m *ContainerManager) StopContainer(ctx context.Context, containerID string) error {
	timeout := 10
	stopOptions := client.ContainerStopOptions{
		Timeout: &timeout,
	}
	_, err := m.cli.ContainerStop(ctx, containerID, stopOptions)
	if err != nil {
		return fmt.Errorf("failed to stop container %s: %w", containerID, err)
	}
	return nil
}

func (m *ContainerManager) StartContainer(ctx context.Context, containerID string) error {
	startOptions := client.ContainerStartOptions{}
	_, err := m.cli.ContainerStart(ctx, containerID, startOptions)
	if err != nil {
		return fmt.Errorf("failed to start container %s: %w", containerID, err)
	}
	return nil
}

func (m *ContainerManager) RestartContainer(ctx context.Context, containerID string) error {
	timeout := 10
	restartOptions := client.ContainerRestartOptions{
		Timeout: &timeout,
	}
	_, err := m.cli.ContainerRestart(ctx, containerID, restartOptions)
	if err != nil {
		return fmt.Errorf("failed to restart container %s: %w", containerID, err)
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
	UpdateStatusLocal     = "local"
)

// isLocalImageMarker is a special marker to indicate the image is local-only
const isLocalImageMarker = "__LOCAL_IMAGE__"

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

	// Check if this is a local-only image (built locally without push to registry)
	if localDigest == isLocalImageMarker {
		info.Status = UpdateStatusLocal
		info.Error = ""
		return []UpdateInfo{info}, nil
	}

	// Sprawdź zdalny digest :latest
	remoteDigest, err := m.getRemoteImageDigest(ctx, latestImage)
	if err != nil {
		if isRateLimitError(err) {
			info.Status = UpdateStatusRateLimit
			info.Error = err.Error()
		} else {
			// If we can't get remote digest and it's not rate limit,
			// the image might be local-only (not pushed to any registry)
			info.Status = UpdateStatusLocal
			info.Error = ""
		}
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
	inspect, err := m.cli.ContainerInspect(ctx, containerID, client.ContainerInspectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to inspect container: %w", err)
	}

	containerName := strings.TrimPrefix(inspect.Container.Name, "/")
	imageName := ""

	if inspect.Container.Config != nil {
		imageName = strings.TrimSpace(inspect.Container.Config.Image)
	}

	result := []UpdateResult{{Container: containerName}}

	if imageName == "" {
		result[0].Success = false
		result[0].Message = "container image is empty"
		return result, nil
	}

	pullResp, err := m.cli.ImagePull(ctx, imageName, client.ImagePullOptions{})
	if err != nil {
		result[0].Success = false
		result[0].Message = fmt.Sprintf("failed to pull image: %v", err)
		return result, nil
	}
	_, _ = io.Copy(io.Discard, pullResp)
	pullResp.Close()

	imageInspect, err := m.cli.ImageInspect(ctx, imageName) // placeholder
	if err != nil {
		result[0].Success = false
		result[0].Message = fmt.Sprintf("failed to inspect pulled image: %v", err)
		return result, nil
	}

	if imageInspect.ID == inspect.Container.Image {
		result[0].Success = true
		result[0].Message = "Container is already up to date"
		return result, nil
	}

	stopOptions := client.ContainerStopOptions{
		Timeout: func() *int { t := 10; return &t }(),
	}
	_, err = m.cli.ContainerStop(ctx, containerID, stopOptions)
	if err != nil {
		result[0].Success = false
		result[0].Message = fmt.Sprintf("failed to stop: %v", err)
		return result, nil
	}

	removeOptions := client.ContainerRemoveOptions{
		Force: true,
	}
	_, err = m.cli.ContainerRemove(ctx, containerID, removeOptions)
	if err != nil {
		result[0].Success = false
		result[0].Message = fmt.Sprintf("failed to remove: %v", err)
		return result, nil
	}

	config := inspect.Container.Config
	hostConfig := inspect.Container.HostConfig

	createOptions := client.ContainerCreateOptions{
		Config:     config,
		HostConfig: hostConfig,
		Name:       containerName,
	}

	createResp, err := m.cli.ContainerCreate(ctx, createOptions)
	if err != nil {
		result[0].Success = false
		result[0].Message = fmt.Sprintf("failed to recreate: %v", err)
		return result, nil
	}

	startOptions := client.ContainerStartOptions{}
	_, err = m.cli.ContainerStart(ctx, createResp.ID, startOptions)
	if err != nil {
		result[0].Success = false
		result[0].Message = fmt.Sprintf("failed to start: %v", err)
		return result, nil
	}

	result[0].Success = true
	result[0].Message = "Container updated successfully"
	return result, nil
}

func (m *ContainerManager) CheckComposeUpdates(ctx context.Context, projectName string) ([]UpdateInfo, error) {
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
	hasLocalImage := false
	checkedImageCount := 0
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

		// Skip local-only images (built locally without push to registry)
		if localDigest == isLocalImageMarker {
			hasLocalImage = true
			continue
		}

		checkedImageCount++
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
	} else if hasLocalImage && checkedImageCount == 0 {
		// All images in the group are local-only
		info.Status = UpdateStatusLocal
	}

	return []UpdateInfo{info}, nil
}

func (m *ContainerManager) getLocalImageDigest(ctx context.Context, image string) (string, error) {
	inspect, err := m.cli.ImageInspect(ctx, image)
	if err != nil {
		return "", fmt.Errorf("failed to inspect local image: %w", err)
	}

	// Local images built without push to registry have empty RepoDigests
	if len(inspect.RepoDigests) == 0 {
		return isLocalImageMarker, nil
	}

	// Check if image has Identity information from a registry
	// Local images don't have Identity.Pull set, remote images do
	if inspect.Identity == nil || len(inspect.Identity.Pull) == 0 {
		// No identity information - this is a local-only image
		return isLocalImageMarker, nil
	}

	// Has identity from registry - find matching digest
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

	// If we have RepoDigests but none match, treat as local image
	return isLocalImageMarker, nil
}

func (m *ContainerManager) getRemoteImageDigest(ctx context.Context, image string) (string, error) {
	m.cacheMu.RLock()
	if cached, ok := m.remoteDigestCache[image]; ok {
		if time.Now().Before(cached.expiresAt) {
			m.cacheMu.RUnlock()
			if cached.errMsg != "" {
				return "", fmt.Errorf("%s", cached.errMsg)
			}
			return cached.digest, nil
		}
	}
	m.cacheMu.RUnlock()

	inspectResult, err := m.cli.DistributionInspect(ctx, image, client.DistributionInspectOptions{})
	if err != nil {
		errMsg := fmt.Sprintf("failed to inspect remote digest: %v", err)
		ttl := cacheTTLError
		if isRateLimitError(err) {
			ttl = cacheTTLRateLimit
		}
		m.cacheRemoteDigest(image, "", errMsg, ttl)
		return "", fmt.Errorf("%s", errMsg)
	}

	remoteDigest := strings.TrimSpace(inspectResult.Descriptor.Digest.String())
	if remoteDigest == "" {
		errMsg := "remote digest unavailable"
		m.cacheRemoteDigest(image, "", errMsg, cacheTTLError)
		return "", fmt.Errorf("%s", errMsg)
	}

	m.cacheRemoteDigest(image, remoteDigest, "", cacheTTLSuccess)
	return remoteDigest, nil
}

func (m *ContainerManager) cacheRemoteDigest(image, digest, errMsg string, ttl time.Duration) {
	m.cacheMu.Lock()
	now := time.Now()
	for key, cached := range m.remoteDigestCache {
		if now.After(cached.expiresAt) {
			delete(m.remoteDigestCache, key)
		}
	}
	m.remoteDigestCache[image] = cachedDigestResult{
		digest:    digest,
		errMsg:    errMsg,
		expiresAt: now.Add(ttl),
	}
	m.cacheMu.Unlock()
}

func (m *ContainerManager) UpdateComposeGroup(ctx context.Context, projectName string) ([]UpdateResult, error) {
	filterArgs := client.Filters{}
	filterArgs = filterArgs.Add("label", fmt.Sprintf("com.docker.compose.project=%s", projectName))

	containers, err := m.cli.ContainerList(ctx, client.ContainerListOptions{
		All:     true,
		Filters: filterArgs,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list containers for project %s: %w", projectName, err)
	}

	if len(containers.Items) == 0 {
		return []UpdateResult{{
			Container: projectName,
			Success:   false,
			Message:   "no containers found in project",
		}}, nil
	}

	var results []UpdateResult
	var mu sync.Mutex

	var wg sync.WaitGroup
	pullResults := make(map[string]error)

	for _, container := range containers.Items {
		wg.Add(1)
		go func(cID, cImage string) {
			defer wg.Done()
			if cImage == "" {
				return
			}
			pullResp, pullErr := m.cli.ImagePull(ctx, cImage, client.ImagePullOptions{})
			if pullErr != nil {
				mu.Lock()
				pullResults[cID] = pullErr
				mu.Unlock()
				return
			}
			defer pullResp.Close()
			_, _ = io.Copy(io.Discard, pullResp)
		}(container.ID, container.Image)
	}
	wg.Wait()

	for _, c := range containers.Items {
		inspect, err := m.cli.ContainerInspect(ctx, c.ID, client.ContainerInspectOptions{})
		if err != nil {
			results = append(results, UpdateResult{Container: strings.TrimPrefix(c.Names[0], "/"), Success: false, Message: fmt.Sprintf("failed to inspect container: %v", err)})
			continue
		}

		containerName := strings.TrimPrefix(inspect.Container.Name, "/")
		imageName := ""
		if inspect.Container.Config != nil {
			imageName = strings.TrimSpace(inspect.Container.Config.Image)
		}
		if imageName == "" {
			results = append(results, UpdateResult{Container: containerName, Success: false, Message: "container image is empty"})
			continue
		}

		if pullErr, ok := pullResults[c.ID]; ok && pullErr != nil {
			results = append(results, UpdateResult{Container: containerName, Success: false, Message: fmt.Sprintf("failed to pull image: %v", pullErr)})
			continue
		}

		imageInspect, err := m.cli.ImageInspect(ctx, imageName) // placeholder
		if err != nil {
			results = append(results, UpdateResult{Container: containerName, Success: false, Message: fmt.Sprintf("failed to inspect pulled image: %v", err)})
			continue
		}

		if imageInspect.ID == inspect.Container.Image {
			results = append(results, UpdateResult{Container: containerName, Success: true, Message: "Container is already up to date"})
			continue
		}

		stopOptions := client.ContainerStopOptions{
			Timeout: func() *int { t := 10; return &t }(),
		}
		_, err = m.cli.ContainerStop(ctx, c.ID, stopOptions)
		if err != nil {
			results = append(results, UpdateResult{Container: containerName, Success: false, Message: fmt.Sprintf("failed to stop: %v", err)})
			continue
		}

		removeOptions := client.ContainerRemoveOptions{Force: true}
		_, err = m.cli.ContainerRemove(ctx, c.ID, removeOptions)
		if err != nil {
			results = append(results, UpdateResult{Container: containerName, Success: false, Message: fmt.Sprintf("failed to remove: %v", err)})
			continue
		}

		createOptions := client.ContainerCreateOptions{
			Config:     inspect.Container.Config,
			HostConfig: inspect.Container.HostConfig,
			Name:       containerName,
		}
		createResp, err := m.cli.ContainerCreate(ctx, createOptions)
		if err != nil {
			results = append(results, UpdateResult{Container: containerName, Success: false, Message: fmt.Sprintf("failed to recreate: %v", err)})
			continue
		}

		startOptions := client.ContainerStartOptions{}
		_, err = m.cli.ContainerStart(ctx, createResp.ID, startOptions)
		if err != nil {
			results = append(results, UpdateResult{Container: containerName, Success: false, Message: fmt.Sprintf("failed to start: %v", err)})
			continue
		}

		results = append(results, UpdateResult{Container: containerName, Success: true, Message: "Container updated successfully"})
	}

	return results, nil
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

func (m *ContainerManager) StopComposeGroup(ctx context.Context, projectName string) error {
	filterArgs := client.Filters{}
	filterArgs = filterArgs.Add("label", fmt.Sprintf("com.docker.compose.project=%s", projectName))

	containers, err := m.cli.ContainerList(ctx, client.ContainerListOptions{
		All:     true,
		Filters: filterArgs,
	})
	if err != nil {
		return fmt.Errorf("failed to list containers for project %s: %w", projectName, err)
	}

	var firstErr error
	for _, c := range containers.Items {
		stopOptions := client.ContainerStopOptions{
			Timeout: func() *int { t := 10; return &t }(),
		}
		_, err := m.cli.ContainerStop(ctx, c.ID, stopOptions)
		if err != nil && firstErr == nil {
			firstErr = fmt.Errorf("failed to stop container %s: %w", c.ID, err)
		}
	}
	return firstErr
}

func (m *ContainerManager) StartComposeGroup(ctx context.Context, projectName string) error {
	filterArgs := client.Filters{}
	filterArgs = filterArgs.Add("label", fmt.Sprintf("com.docker.compose.project=%s", projectName))

	containers, err := m.cli.ContainerList(ctx, client.ContainerListOptions{
		All:     true,
		Filters: filterArgs,
	})
	if err != nil {
		return fmt.Errorf("failed to list containers for project %s: %w", projectName, err)
	}

	var firstErr error
	for _, c := range containers.Items {
		startOptions := client.ContainerStartOptions{}
		_, err := m.cli.ContainerStart(ctx, c.ID, startOptions)
		if err != nil && firstErr == nil {
			firstErr = fmt.Errorf("failed to start container %s: %w", c.ID, err)
		}
	}
	return firstErr
}

func (m *ContainerManager) RestartComposeGroup(ctx context.Context, projectName string) error {
	filterArgs := client.Filters{}
	filterArgs = filterArgs.Add("label", fmt.Sprintf("com.docker.compose.project=%s", projectName))

	containers, err := m.cli.ContainerList(ctx, client.ContainerListOptions{
		All:     true,
		Filters: filterArgs,
	})
	if err != nil {
		return fmt.Errorf("failed to list containers for project %s: %w", projectName, err)
	}

	var firstErr error
	for _, c := range containers.Items {
		restartOptions := client.ContainerRestartOptions{
			Timeout: func() *int { t := 10; return &t }(),
		}
		_, err := m.cli.ContainerRestart(ctx, c.ID, restartOptions)
		if err != nil && firstErr == nil {
			firstErr = fmt.Errorf("failed to restart container %s: %w", c.ID, err)
		}
	}
	return firstErr
}
