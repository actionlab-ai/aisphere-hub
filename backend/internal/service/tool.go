package service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strconv"
	"strings"

	"github.com/actionlab-ai/aisphere-hub/backend/internal/httputil"
	"github.com/actionlab-ai/aisphere-hub/backend/internal/model"
)

func (s *Service) CreateTool(namespaceID string, req model.ToolUpsertRequest, ownerSubject string) (*model.ToolRecord, error) {
	if err := validateToolRequest(req, false); err != nil {
		return nil, err
	}
	var created *model.ToolRecord
	err := s.store.WithWrite(func() error {
		existing, err := s.store.LoadTool(namespaceID, req.ID)
		if err != nil {
			return err
		}
		if existing != nil {
			return httputil.Conflict("tool already exists: " + req.ID)
		}
		now := model.NowMillis()
		version := firstNonEmptyAgent(req.Version, "1.0.0")
		status, err := normalizedResourceStatus(req.Status)
		if err != nil {
			return err
		}
		definition := normalizedToolDefinition(req.Definition)
		created = &model.ToolRecord{
			NamespaceID: namespaceID, App: model.DefaultApp, OwnerSubject: strings.TrimSpace(ownerSubject),
			ID: req.ID, DisplayName: strings.TrimSpace(req.DisplayName), Description: strings.TrimSpace(req.Description),
			Status: status, Scope: firstNonEmptyAgent(req.Scope, model.ScopePrivate),
			Labels: normalizedResourceLabels(req.Labels, version), LatestVersion: version, CreateTime: now, UpdateTime: now,
			Versions: map[string]*model.ToolVersionRecord{version: newToolVersion(version, definition, ownerSubject, req.CommitMsg, now)},
		}
		if err := s.store.SaveTool(created); err != nil {
			return err
		}
		s.emitCatalogEvent(model.CatalogEventToolUpdated, "tool", created.ID, "", version, created.Versions[version].Revision, map[string]interface{}{"status": created.Status, "latest": created.LatestVersion})
		return nil
	})
	return created, err
}

func (s *Service) GetTool(namespaceID, id string) (*model.ToolRecord, error) {
	if strings.TrimSpace(id) == "" {
		return nil, httputil.BadRequest("tool id is required")
	}
	rec, err := s.store.LoadTool(namespaceID, strings.TrimSpace(id))
	if err != nil {
		return nil, err
	}
	if rec == nil {
		return nil, httputil.NotFound("tool not found: " + id)
	}
	return rec, nil
}

func (s *Service) ListTools(namespaceID string) ([]*model.ToolRecord, error) {
	return s.store.ListTools(namespaceID)
}

func (s *Service) UpdateTool(namespaceID, id string, req model.ToolUpsertRequest, operator string) (*model.ToolRecord, error) {
	id = strings.TrimSpace(id)
	if req.ID != "" && req.ID != id {
		return nil, httputil.BadRequest("tool id cannot be changed")
	}
	req.ID = id
	if err := validateToolRequest(req, false); err != nil {
		return nil, err
	}
	var updated *model.ToolRecord
	err := s.store.WithWrite(func() error {
		rec, err := s.store.LoadTool(namespaceID, id)
		if err != nil {
			return err
		}
		if rec == nil {
			return httputil.NotFound("tool not found: " + id)
		}
		version := firstNonEmptyAgent(req.Version, nextToolVersion(rec.LatestVersion))
		if rec.Versions[version] != nil {
			return httputil.Conflict("tool version already exists: " + version)
		}
		rec.DisplayName = strings.TrimSpace(req.DisplayName)
		rec.Description = strings.TrimSpace(req.Description)
		if strings.TrimSpace(req.Status) != "" {
			status, err := normalizedResourceStatus(req.Status)
			if err != nil {
				return err
			}
			rec.Status = status
		}
		if strings.TrimSpace(req.Scope) != "" {
			rec.Scope = strings.TrimSpace(req.Scope)
		}
		now := model.NowMillis()
		rec.UpdateTime = now
		rec.LatestVersion = version
		rec.Labels = mergeResourceLabels(rec.Labels, req.Labels, version)
		rec.Versions[version] = newToolVersion(version, normalizedToolDefinition(req.Definition), operator, req.CommitMsg, now)
		if err := s.store.SaveTool(rec); err != nil {
			return err
		}
		s.emitCatalogEvent(model.CatalogEventToolUpdated, "tool", rec.ID, "", version, rec.Versions[version].Revision, map[string]interface{}{"status": rec.Status, "latest": rec.LatestVersion})
		updated = rec
		return nil
	})
	return updated, err
}

func (s *Service) DeleteTool(namespaceID, id string) error {
	if strings.TrimSpace(id) == "" {
		return httputil.BadRequest("tool id is required")
	}
	return s.store.WithWrite(func() error {
		rec, err := s.store.LoadTool(namespaceID, strings.TrimSpace(id))
		if err != nil {
			return err
		}
		if rec == nil {
			return httputil.NotFound("tool not found: " + id)
		}
		if err := s.store.DeleteTool(namespaceID, id); err != nil {
			return err
		}
		s.emitCatalogEvent(model.CatalogEventToolDeleted, "tool", id, "", rec.LatestVersion, "", nil)
		return nil
	})
}

func (s *Service) GetToolVersion(namespaceID, id, version string) (*model.ToolVersionRecord, error) {
	rec, err := s.GetTool(namespaceID, id)
	if err != nil {
		return nil, err
	}
	version = strings.TrimSpace(version)
	if version == "" {
		version = rec.LatestVersion
	}
	vr := rec.Versions[version]
	if vr == nil {
		return nil, httputil.NotFound("tool version not found: " + version)
	}
	return vr, nil
}

func (s *Service) ReportToolFailure(namespaceID string, req model.ToolFailureReportRequest, reporter, object string) (*model.ToolFailureRecord, error) {
	if strings.TrimSpace(req.ToolID) == "" {
		return nil, httputil.BadRequest("tool id is required")
	}
	if strings.TrimSpace(req.ErrorMessage) == "" {
		return nil, httputil.BadRequest("error message is required")
	}
	now := model.NowMillis()
	rec := &model.ToolFailureRecord{
		ID:  "tf_" + strconv.FormatInt(now, 10) + "_" + randomFailureSuffix(req.ToolID+req.SessionID+req.TraceID),
		App: model.DefaultApp, NamespaceID: namespaceID, Object: strings.TrimSpace(object),
		ToolID: strings.TrimSpace(req.ToolID), ToolVersion: strings.TrimSpace(req.ToolVersion),
		AgentID: strings.TrimSpace(req.AgentID), AgentVersion: strings.TrimSpace(req.AgentVersion),
		RuntimeID: strings.TrimSpace(req.RuntimeID), SessionID: strings.TrimSpace(req.SessionID), RunID: strings.TrimSpace(req.RunID), TraceID: strings.TrimSpace(req.TraceID), SnapshotID: strings.TrimSpace(req.SnapshotID),
		Attempt: req.Attempt, ErrorCode: strings.TrimSpace(req.ErrorCode), ErrorMessage: strings.TrimSpace(req.ErrorMessage),
		Retryable: req.Retryable, InputDigest: strings.TrimSpace(req.InputDigest), InputPreview: strings.TrimSpace(req.InputPreview), DurationMillis: req.DurationMillis, Metadata: req.Metadata, Reporter: strings.TrimSpace(reporter), CreateTime: now,
	}
	err := s.store.WithWrite(func() error { return s.store.AppendToolFailure(rec) })
	return rec, err
}

func (s *Service) ListToolFailures(q model.ToolFailureQuery) ([]*model.ToolFailureRecord, int64, error) {
	return s.store.ListToolFailures(q)
}

func newToolVersion(version string, definition model.ToolDefinition, author, commitMsg string, now int64) *model.ToolVersionRecord {
	b, _ := json.Marshal(definition)
	sum := sha256.Sum256(b)
	digest := hex.EncodeToString(sum[:])
	return &model.ToolVersionRecord{Version: version, Revision: digest[:16], SHA256: digest, Author: strings.TrimSpace(author), CommitMsg: commitMsg, CreateTime: now, Definition: definition}
}

func validateToolRequest(req model.ToolUpsertRequest, allowEmptyDefinition bool) error {
	if strings.TrimSpace(req.ID) == "" {
		return httputil.BadRequest("tool id is required")
	}
	for _, r := range req.ID {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' || r == '.') {
			return httputil.BadRequest("tool id may contain only letters, numbers, dot, hyphen, and underscore")
		}
	}
	if strings.TrimSpace(req.Version) != "" && !isAgentVersion(req.Version) {
		return httputil.BadRequest("tool version must use numeric semantic version format")
	}
	if allowEmptyDefinition && strings.TrimSpace(req.Definition.Runtime.Type) == "" {
		return nil
	}
	definition := normalizedToolDefinition(req.Definition)
	if definition.Runtime.Type == "" {
		return httputil.BadRequest("tool definition runtime.type is required")
	}
	switch definition.Runtime.Type {
	case "mcp":
		if definition.Runtime.Server == "" || definition.Runtime.Name == "" {
			return httputil.BadRequest("mcp tool requires runtime.server and runtime.name")
		}
	case "http", "openapi":
		if definition.Runtime.URL == "" {
			return httputil.BadRequest(definition.Runtime.Type + " tool requires runtime.url")
		}
		if !isAllowedHTTPMethod(definition.Runtime.Method) {
			return httputil.BadRequest("http/openapi tool runtime.method must be GET, POST, PUT, PATCH, or DELETE")
		}
	case "builtin", "function":
		if definition.Runtime.Name == "" && definition.Runtime.EntryPoint == "" {
			return httputil.BadRequest(definition.Runtime.Type + " tool requires runtime.name or runtime.entryPoint")
		}
	default:
		return httputil.BadRequest("unsupported tool runtime.type: " + definition.Runtime.Type)
	}
	if err := validateToolExecutionDefinition(definition.Execution); err != nil {
		return err
	}
	if err := validateToolSecretReferences(definition.Runtime); err != nil {
		return err
	}
	if err := validateToolExecutionSecrets(definition.Execution); err != nil {
		return err
	}
	return nil
}

func normalizedToolDefinition(in model.ToolDefinition) model.ToolDefinition {
	out := in
	out.Runtime.Type = strings.ToLower(strings.TrimSpace(in.Runtime.Type))
	out.Runtime.Server = strings.TrimSpace(in.Runtime.Server)
	out.Runtime.Name = strings.TrimSpace(in.Runtime.Name)
	out.Runtime.URL = strings.TrimSpace(in.Runtime.URL)
	out.Runtime.Method = strings.ToUpper(strings.TrimSpace(in.Runtime.Method))
	out.Runtime.Package = strings.TrimSpace(in.Runtime.Package)
	out.Runtime.EntryPoint = strings.TrimSpace(in.Runtime.EntryPoint)
	out.Runtime.CredentialRef = strings.TrimSpace(in.Runtime.CredentialRef)
	out.Execution = normalizedToolExecutionDefinition(in.Execution, out.Runtime.Type)
	if out.Runtime.Method == "" && (out.Runtime.Type == "http" || out.Runtime.Type == "openapi") {
		out.Runtime.Method = "POST"
	}
	if out.TimeoutMillis <= 0 {
		out.TimeoutMillis = 30000
	}
	if out.Execution.Resources != nil && out.Execution.Resources.TimeoutMillis <= 0 {
		out.Execution.Resources.TimeoutMillis = out.TimeoutMillis
	}
	return out
}

func normalizedToolExecutionDefinition(in model.ToolExecutionDefinition, runtimeType string) model.ToolExecutionDefinition {
	out := in
	out.Placement = strings.ToLower(strings.TrimSpace(out.Placement))
	out.Runner = strings.ToLower(strings.TrimSpace(out.Runner))
	out.Image = strings.TrimSpace(out.Image)
	out.Command = strings.TrimSpace(out.Command)
	out.WorkingDir = strings.TrimSpace(out.WorkingDir)
	out.Filesystem = strings.ToLower(strings.TrimSpace(out.Filesystem))
	out.Network = strings.ToLower(strings.TrimSpace(out.Network))
	for i := range out.Mounts {
		out.Mounts[i].Name = strings.TrimSpace(out.Mounts[i].Name)
		out.Mounts[i].Ref = strings.TrimSpace(out.Mounts[i].Ref)
		out.Mounts[i].MountPath = strings.TrimSpace(out.Mounts[i].MountPath)
		out.Mounts[i].Mode = strings.ToLower(strings.TrimSpace(out.Mounts[i].Mode))
		if out.Mounts[i].Mode == "" {
			out.Mounts[i].Mode = "ro"
		}
	}
	if out.Placement == "" {
		switch runtimeType {
		case "http", "openapi":
			out.Placement = model.ToolExecutionPlacementRemote
		case "mcp":
			out.Placement = model.ToolExecutionPlacementRemote
		default:
			out.Placement = model.ToolExecutionPlacementSandbox
		}
	}
	if out.Runner == "" {
		out.Runner = runtimeType
	}
	if out.Filesystem == "" {
		switch out.Placement {
		case model.ToolExecutionPlacementSandbox:
			out.Filesystem = model.ToolFilesystemWorkspace
		default:
			out.Filesystem = model.ToolFilesystemNone
		}
	}
	if out.Network == "" {
		switch out.Placement {
		case model.ToolExecutionPlacementSandbox:
			out.Network = model.ToolNetworkNone
		case model.ToolExecutionPlacementRemote:
			out.Network = model.ToolNetworkRestricted
		default:
			out.Network = model.ToolNetworkNone
		}
	}
	return out
}

func validateToolExecutionDefinition(exec model.ToolExecutionDefinition) error {
	switch exec.Placement {
	case "", model.ToolExecutionPlacementSandbox, model.ToolExecutionPlacementRuntime, model.ToolExecutionPlacementRemote, model.ToolExecutionPlacementHub:
	default:
		return httputil.BadRequest("tool execution.placement must be sandbox, runtime, remote, or hub")
	}
	switch exec.Filesystem {
	case "", model.ToolFilesystemNone, model.ToolFilesystemReadonly, model.ToolFilesystemWorkspace:
	default:
		return httputil.BadRequest("tool execution.filesystem must be none, readonly, or workspace")
	}
	switch exec.Network {
	case "", model.ToolNetworkNone, model.ToolNetworkRestricted, model.ToolNetworkEgress:
	default:
		return httputil.BadRequest("tool execution.network must be none, restricted, or egress")
	}
	for _, mount := range exec.Mounts {
		if mount.Name == "" || mount.Ref == "" || mount.MountPath == "" {
			return httputil.BadRequest("tool execution.mounts require name, ref, and mountPath")
		}
		if strings.HasPrefix(mount.Ref, "/") || strings.Contains(mount.Ref, "..") {
			return httputil.BadRequest("tool execution.mounts.ref must be a logical reference, not a host path")
		}
		if !strings.HasPrefix(mount.MountPath, "/") || mount.MountPath == "/" || strings.Contains(mount.MountPath, "..") {
			return httputil.BadRequest("tool execution.mounts.mountPath must be an absolute sandbox path and cannot be root or contain ..")
		}
		if mount.Mode != "" && mount.Mode != "ro" && mount.Mode != "rw" {
			return httputil.BadRequest("tool execution.mounts.mode must be ro or rw")
		}
	}
	return nil
}

func validateToolExecutionSecrets(exec model.ToolExecutionDefinition) error {
	for key, value := range exec.Env {
		if isSensitiveHeaderName(key) && !isSecretPlaceholder(value) {
			return httputil.BadRequest("tool execution.env contains a sensitive value; use secretRefs or ${secret:name}: " + key)
		}
	}
	for _, ref := range exec.SecretRefs {
		ref = strings.TrimSpace(ref)
		if ref == "" {
			continue
		}
		if !isSafeSecretRef(ref) {
			return httputil.BadRequest("tool execution.secretRefs must use secret://, vault://, env://, or project-secret:// references")
		}
	}
	return nil
}

func isAllowedHTTPMethod(method string) bool {
	switch strings.ToUpper(strings.TrimSpace(method)) {
	case "GET", "POST", "PUT", "PATCH", "DELETE":
		return true
	default:
		return false
	}
}

func validateToolSecretReferences(runtime model.ToolRuntimeDefinition) error {
	if runtime.CredentialRef != "" && !isSafeToolRef(runtime.CredentialRef) {
		return httputil.BadRequest("tool runtime.credentialRef contains unsafe characters")
	}
	for key, value := range runtime.Headers {
		if !isSensitiveHeaderName(key) {
			continue
		}
		value = strings.TrimSpace(value)
		if value == "" || strings.HasPrefix(value, "${secret:") || strings.HasPrefix(value, "secret://") {
			continue
		}
		return httputil.BadRequest("sensitive header " + key + " must use runtime.credentialRef or a secret reference placeholder")
	}
	return nil
}

func isSensitiveHeaderName(key string) bool {
	k := strings.ToLower(strings.TrimSpace(key))
	return k == "authorization" || k == "proxy-authorization" || k == "cookie" || strings.Contains(k, "token") || strings.Contains(k, "api-key") || strings.Contains(k, "apikey") || strings.Contains(k, "secret")
}

func isSecretPlaceholder(value string) bool {
	v := strings.TrimSpace(value)
	return strings.HasPrefix(v, "${secret:") || strings.HasPrefix(v, "secret://") || strings.HasPrefix(v, "vault://") || strings.HasPrefix(v, "env://") || strings.HasPrefix(v, "project-secret://")
}

func isSafeSecretRef(ref string) bool {
	ref = strings.TrimSpace(ref)
	if !(strings.HasPrefix(ref, "secret://") || strings.HasPrefix(ref, "vault://") || strings.HasPrefix(ref, "env://") || strings.HasPrefix(ref, "project-secret://")) {
		return false
	}
	return isSafeToolRef(ref)
}

func isSafeToolRef(ref string) bool {
	for _, r := range ref {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' || r == '.' || r == '/' || r == ':') {
			return false
		}
	}
	return true
}

func nextToolVersion(current string) string {
	parts := strings.Split(current, ".")
	if len(parts) != 3 {
		return "1.0.0"
	}
	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return "1.0.0"
	}
	return parts[0] + "." + parts[1] + "." + strconv.Itoa(patch+1)
}

func randomFailureSuffix(seed string) string {
	sum := sha256.Sum256([]byte(seed + strconv.FormatInt(model.NowMillis(), 10)))
	return hex.EncodeToString(sum[:])[:8]
}

func SortToolsByUpdatedDesc(records []*model.ToolRecord) {
	sort.Slice(records, func(i, j int) bool { return records[i].UpdateTime > records[j].UpdateTime })
}
