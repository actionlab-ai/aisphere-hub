package store

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/actionlab-ai/aisphere-hub/backend/internal/model"
)

type Store struct {
	mu   sync.RWMutex
	root string
}

func New(root string) (*Store, error) {
	if root == "" {
		root = "data/aihub"
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, err
	}
	return &Store{root: root}, nil
}

func (s *Store) Root() string { return s.root }

func (s *Store) WithWrite(fn func() error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return fn()
}

func (s *Store) WithRead(fn func() error) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return fn()
}

func (s *Store) Load(namespaceID, name string) (*model.SkillRecord, error) {
	path := s.path(namespaceID, name)
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var rec model.SkillRecord
	if err := json.Unmarshal(b, &rec); err != nil {
		return nil, err
	}
	if rec.Labels == nil {
		rec.Labels = map[string]string{}
	}
	if rec.Versions == nil {
		rec.Versions = map[string]*model.VersionRecord{}
	}
	return &rec, nil
}

func (s *Store) Save(rec *model.SkillRecord) error {
	if rec == nil {
		return nil
	}
	if rec.Labels == nil {
		rec.Labels = map[string]string{}
	}
	if rec.Versions == nil {
		rec.Versions = map[string]*model.VersionRecord{}
	}
	path := s.path(rec.NamespaceID, rec.Name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	b, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func (s *Store) Delete(namespaceID, name string) error {
	return os.RemoveAll(filepath.Dir(s.path(namespaceID, name)))
}

func (s *Store) List(namespaceID string) ([]*model.SkillRecord, error) {
	dir := filepath.Join(s.root, "skills")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []*model.SkillRecord{}, nil
		}
		return nil, err
	}
	out := make([]*model.SkillRecord, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		rec, err := s.Load(model.DefaultNamespace, unsafename(e.Name()))
		if err != nil {
			return nil, err
		}
		if rec != nil {
			out = append(out, rec)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UpdateTime > out[j].UpdateTime })
	return out, nil
}

func (s *Store) path(namespaceID, name string) string {
	return filepath.Join(s.root, "skills", safe(name), "meta.json")
}

func (s *Store) LoadAgent(namespaceID, id string) (*model.AgentRecord, error) {
	b, err := os.ReadFile(s.agentPath(namespaceID, id))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var rec model.AgentRecord
	if err := json.Unmarshal(b, &rec); err != nil {
		return nil, err
	}
	if rec.NamespaceID == "" {
		rec.NamespaceID = namespaceID
	}
	if rec.ID == "" {
		rec.ID = id
	}
	if rec.Labels == nil {
		rec.Labels = map[string]string{}
	}
	if rec.Versions == nil {
		rec.Versions = map[string]*model.AgentVersionRecord{}
	}
	return &rec, nil
}

func (s *Store) SaveAgent(rec *model.AgentRecord) error {
	if rec == nil {
		return nil
	}
	if rec.Labels == nil {
		rec.Labels = map[string]string{}
	}
	if rec.Versions == nil {
		rec.Versions = map[string]*model.AgentVersionRecord{}
	}
	path := s.agentPath(rec.NamespaceID, rec.ID)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func (s *Store) DeleteAgent(namespaceID, id string) error {
	return os.RemoveAll(filepath.Dir(s.agentPath(namespaceID, id)))
}

func (s *Store) ListAgents(namespaceID string) ([]*model.AgentRecord, error) {
	dir := filepath.Join(s.root, "agents")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []*model.AgentRecord{}, nil
		}
		return nil, err
	}
	out := make([]*model.AgentRecord, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		rec, err := s.LoadAgent(namespaceID, unsafename(e.Name()))
		if err != nil {
			return nil, err
		}
		if rec != nil {
			out = append(out, rec)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UpdateTime > out[j].UpdateTime })
	return out, nil
}

func (s *Store) agentPath(namespaceID, id string) string {
	return filepath.Join(s.root, "agents", safe(id), "meta.json")
}

func (s *Store) LoadTool(namespaceID, id string) (*model.ToolRecord, error) {
	b, err := os.ReadFile(s.toolPath(namespaceID, id))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var rec model.ToolRecord
	if err := json.Unmarshal(b, &rec); err != nil {
		return nil, err
	}
	if rec.NamespaceID == "" {
		rec.NamespaceID = namespaceID
	}
	if rec.ID == "" {
		rec.ID = id
	}
	if rec.Labels == nil {
		rec.Labels = map[string]string{}
	}
	if rec.Versions == nil {
		rec.Versions = map[string]*model.ToolVersionRecord{}
	}
	return &rec, nil
}

func (s *Store) SaveTool(rec *model.ToolRecord) error {
	if rec == nil {
		return nil
	}
	if rec.Labels == nil {
		rec.Labels = map[string]string{}
	}
	if rec.Versions == nil {
		rec.Versions = map[string]*model.ToolVersionRecord{}
	}
	path := s.toolPath(rec.NamespaceID, rec.ID)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func (s *Store) DeleteTool(namespaceID, id string) error {
	return os.RemoveAll(filepath.Dir(s.toolPath(namespaceID, id)))
}

func (s *Store) ListTools(namespaceID string) ([]*model.ToolRecord, error) {
	dir := filepath.Join(s.root, "tools")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []*model.ToolRecord{}, nil
		}
		return nil, err
	}
	out := make([]*model.ToolRecord, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		rec, err := s.LoadTool(namespaceID, unsafename(e.Name()))
		if err != nil {
			return nil, err
		}
		if rec != nil {
			out = append(out, rec)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UpdateTime > out[j].UpdateTime })
	return out, nil
}

func (s *Store) toolPath(namespaceID, id string) string {
	return filepath.Join(s.root, "tools", safe(id), "meta.json")
}

func (s *Store) toolFailuresDir() string {
	return filepath.Join(s.root, "_tool_failures")
}

func (s *Store) AppendToolFailure(f *model.ToolFailureRecord) error {
	if f == nil {
		return nil
	}
	if f.ID == "" {
		f.ID = strconv.FormatInt(model.NowMillis(), 10) + "-" + safe(f.ToolID)
	}
	if f.CreateTime == 0 {
		f.CreateTime = model.NowMillis()
	}
	return writeJSONFile(filepath.Join(s.toolFailuresDir(), safe(f.ID)+".json"), f)
}

func (s *Store) ListToolFailures(q model.ToolFailureQuery) ([]*model.ToolFailureRecord, int64, error) {
	entries, err := os.ReadDir(s.toolFailuresDir())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []*model.ToolFailureRecord{}, 0, nil
		}
		return nil, 0, err
	}
	out := []*model.ToolFailureRecord{}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		var f model.ToolFailureRecord
		if readJSONFile(filepath.Join(s.toolFailuresDir(), e.Name()), &f) != nil {
			continue
		}
		if q.ToolID != "" && f.ToolID != q.ToolID {
			continue
		}
		if q.AgentID != "" && f.AgentID != q.AgentID {
			continue
		}
		if q.RuntimeID != "" && f.RuntimeID != q.RuntimeID {
			continue
		}
		if q.SessionID != "" && f.SessionID != q.SessionID {
			continue
		}
		if q.RunID != "" && f.RunID != q.RunID {
			continue
		}
		if q.TraceID != "" && f.TraceID != q.TraceID {
			continue
		}
		if q.SnapshotID != "" && f.SnapshotID != q.SnapshotID {
			continue
		}
		out = append(out, &f)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreateTime > out[j].CreateTime })
	total := int64(len(out))
	if q.Limit <= 0 || q.Limit > 500 {
		q.Limit = 100
	}
	if len(out) > q.Limit {
		out = out[:q.Limit]
	}
	return out, total, nil
}

func (s *Store) LoadSandboxProfile(namespaceID, id string) (*model.SandboxProfile, error) {
	var p model.SandboxProfile
	if err := readJSONFile(s.sandboxProfilePath(namespaceID, id), &p); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	if p.NamespaceID == "" {
		p.NamespaceID = namespaceID
	}
	if p.ID == "" {
		p.ID = id
	}
	if p.Labels == nil {
		p.Labels = map[string]string{}
	}
	if p.Metadata == nil {
		p.Metadata = map[string]interface{}{}
	}
	return &p, nil
}
func (s *Store) SaveSandboxProfile(p *model.SandboxProfile) error {
	if p == nil {
		return nil
	}
	return writeJSONFile(s.sandboxProfilePath(p.NamespaceID, p.ID), p)
}
func (s *Store) DeleteSandboxProfile(namespaceID, id string) error {
	return os.RemoveAll(filepath.Dir(s.sandboxProfilePath(namespaceID, id)))
}
func (s *Store) ListSandboxProfiles(namespaceID string) ([]*model.SandboxProfile, error) {
	dir := filepath.Join(s.root, "sandbox_profiles")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []*model.SandboxProfile{}, nil
		}
		return nil, err
	}
	out := []*model.SandboxProfile{}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		p, err := s.LoadSandboxProfile(namespaceID, unsafename(e.Name()))
		if err != nil {
			return nil, err
		}
		if p != nil {
			out = append(out, p)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UpdateTime > out[j].UpdateTime })
	return out, nil
}
func (s *Store) sandboxProfilePath(namespaceID, id string) string {
	return filepath.Join(s.root, "sandbox_profiles", safe(id), "profile.json")
}
func (s *Store) LoadSandboxPolicy(namespaceID, id string) (*model.SandboxPolicy, error) {
	var p model.SandboxPolicy
	if err := readJSONFile(s.sandboxPolicyPath(namespaceID, id), &p); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	if p.NamespaceID == "" {
		p.NamespaceID = namespaceID
	}
	if p.ID == "" {
		p.ID = id
	}
	if p.Labels == nil {
		p.Labels = map[string]string{}
	}
	if p.Metadata == nil {
		p.Metadata = map[string]interface{}{}
	}
	return &p, nil
}
func (s *Store) SaveSandboxPolicy(p *model.SandboxPolicy) error {
	if p == nil {
		return nil
	}
	return writeJSONFile(s.sandboxPolicyPath(p.NamespaceID, p.ID), p)
}
func (s *Store) DeleteSandboxPolicy(namespaceID, id string) error {
	return os.RemoveAll(filepath.Dir(s.sandboxPolicyPath(namespaceID, id)))
}
func (s *Store) ListSandboxPolicies(namespaceID string) ([]*model.SandboxPolicy, error) {
	dir := filepath.Join(s.root, "sandbox_policies")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []*model.SandboxPolicy{}, nil
		}
		return nil, err
	}
	out := []*model.SandboxPolicy{}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		p, err := s.LoadSandboxPolicy(namespaceID, unsafename(e.Name()))
		if err != nil {
			return nil, err
		}
		if p != nil {
			out = append(out, p)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UpdateTime > out[j].UpdateTime })
	return out, nil
}
func (s *Store) sandboxPolicyPath(namespaceID, id string) string {
	return filepath.Join(s.root, "sandbox_policies", safe(id), "policy.json")
}

func safe(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "/", "_")
	s = strings.ReplaceAll(s, "\\", "_")
	s = strings.ReplaceAll(s, "..", "_")
	if s == "" {
		return "_"
	}
	return s
}

func unsafename(s string) string { return s }

func (s *Store) LoadGroup(namespaceID, name string) (*model.SkillGroup, error) {
	path := s.groupPath(namespaceID, name)
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var g model.SkillGroup
	if err := json.Unmarshal(b, &g); err != nil {
		return nil, err
	}
	if g.Labels == nil {
		g.Labels = map[string]string{}
	}
	if g.Members == nil {
		g.Members = []model.SkillGroupMember{}
	}
	if g.Metadata == nil {
		g.Metadata = map[string]interface{}{}
	}
	return &g, nil
}

func (s *Store) SaveGroup(g *model.SkillGroup) error {
	if g == nil {
		return nil
	}
	if g.Labels == nil {
		g.Labels = map[string]string{}
	}
	if g.Members == nil {
		g.Members = []model.SkillGroupMember{}
	}
	if g.Metadata == nil {
		g.Metadata = map[string]interface{}{}
	}
	path := s.groupPath(model.DefaultNamespace, g.Name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(g, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func (s *Store) DeleteGroup(namespaceID, name string) error {
	return os.RemoveAll(filepath.Dir(s.groupPath(namespaceID, name)))
}

func (s *Store) ListGroups(namespaceID string) ([]*model.SkillGroup, error) {
	dir := filepath.Join(s.root, "groups")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []*model.SkillGroup{}, nil
		}
		return nil, err
	}
	out := make([]*model.SkillGroup, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		g, err := s.LoadGroup(model.DefaultNamespace, unsafename(e.Name()))
		if err != nil {
			return nil, err
		}
		if g != nil {
			out = append(out, g)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UpdateTime > out[j].UpdateTime })
	return out, nil
}

func (s *Store) groupPath(namespaceID, name string) string {
	return filepath.Join(s.root, "groups", safe(name), "group.json")
}

func (s *Store) proposalPath(proposalID string) string {
	return filepath.Join(s.root, "_proposals", safe(proposalID), "proposal.json")
}

func (s *Store) overlayPath(overlayRef string) string {
	return filepath.Join(s.root, "_overlays", safe(overlayRef), "overlay.json")
}

func (s *Store) validationPath(proposalID string) string {
	return filepath.Join(s.root, "_proposals", safe(proposalID), "validations.json")
}

func (s *Store) SaveProposal(p *model.SkillProposal) error {
	if p == nil {
		return nil
	}
	path := s.proposalPath(p.ProposalID)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func (s *Store) LoadProposal(proposalID string) (*model.SkillProposal, error) {
	b, err := os.ReadFile(s.proposalPath(proposalID))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var p model.SkillProposal
	if err := json.Unmarshal(b, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *Store) ListProposals(q model.ProposalQuery) ([]*model.SkillProposal, int64, error) {
	dir := filepath.Join(s.root, "_proposals")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []*model.SkillProposal{}, 0, nil
		}
		return nil, 0, err
	}
	out := []*model.SkillProposal{}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		p, err := s.LoadProposal(unsafename(e.Name()))
		if err != nil {
			return nil, 0, err
		}
		if p == nil {
			continue
		}
		if q.NamespaceID != "" && p.NamespaceID != q.NamespaceID {
			continue
		}
		if q.SkillName != "" && p.SkillName != q.SkillName {
			continue
		}
		if q.Status != "" && p.Status != q.Status {
			continue
		}
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UpdateTime > out[j].UpdateTime })
	return out, int64(len(out)), nil
}

func (s *Store) SaveOverlay(o *model.SkillOverlay) error {
	if o == nil {
		return nil
	}
	path := s.overlayPath(o.OverlayRef)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(o, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func (s *Store) LoadOverlay(overlayRef string) (*model.SkillOverlay, error) {
	b, err := os.ReadFile(s.overlayPath(overlayRef))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var o model.SkillOverlay
	if err := json.Unmarshal(b, &o); err != nil {
		return nil, err
	}
	return &o, nil
}

func (s *Store) SaveProposalValidation(v *model.ProposalValidation) error {
	if v == nil {
		return nil
	}
	list, _ := s.ListProposalValidations(v.ProposalID)
	list = append(list, v)
	path := s.validationPath(v.ProposalID)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func (s *Store) ListProposalValidations(proposalID string) ([]*model.ProposalValidation, error) {
	b, err := os.ReadFile(s.validationPath(proposalID))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []*model.ProposalValidation{}, nil
		}
		return nil, err
	}
	var out []*model.ProposalValidation
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) namespacePath(namespaceID string) string {
	return filepath.Join(s.root, safe(namespaceID), "_namespace.json")
}
func (s *Store) namespaceMemberPath(namespaceID, subjectID string) string {
	return filepath.Join(s.root, safe(namespaceID), "_members", safe(subjectID)+".json")
}
func (s *Store) socialDir(namespaceID, skillName string) string {
	return filepath.Join(s.root, safe(namespaceID), safe(skillName), "_social")
}
func (s *Store) auditDir() string { return filepath.Join(s.root, "_audit") }
func (s *Store) tokenPath(keyID string) string {
	return filepath.Join(s.root, "_tokens", safe(keyID)+".json")
}
func (s *Store) idempotencyPath(key string) string {
	return filepath.Join(s.root, "_idempotency", safe(key)+".json")
}

func (s *Store) SaveNamespace(ns *model.NamespaceInfo) error {
	if ns == nil {
		return nil
	}
	if ns.NamespaceID == "" {
		ns.NamespaceID = model.DefaultNamespace
	}
	if ns.CreateTime == 0 {
		ns.CreateTime = model.NowMillis()
	}
	ns.UpdateTime = model.NowMillis()
	if ns.Metadata == nil {
		ns.Metadata = map[string]interface{}{}
	}
	path := s.namespacePath(ns.NamespaceID)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(ns, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

func (s *Store) LoadNamespace(namespaceID string) (*model.NamespaceInfo, error) {
	b, err := os.ReadFile(s.namespacePath(namespaceID))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var ns model.NamespaceInfo
	if err := json.Unmarshal(b, &ns); err != nil {
		return nil, err
	}
	if ns.Metadata == nil {
		ns.Metadata = map[string]interface{}{}
	}
	return &ns, nil
}

func (s *Store) ListNamespaces() ([]*model.NamespaceInfo, error) {
	entries, err := os.ReadDir(s.root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []*model.NamespaceInfo{}, nil
		}
		return nil, err
	}
	out := []*model.NamespaceInfo{}
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), "_") {
			continue
		}
		ns, err := s.LoadNamespace(unsafename(e.Name()))
		if err != nil {
			return nil, err
		}
		if ns == nil {
			ns = &model.NamespaceInfo{NamespaceID: unsafename(e.Name()), DisplayName: unsafename(e.Name()), Visibility: model.ScopePrivate, CreateTime: model.NowMillis(), UpdateTime: model.NowMillis(), Metadata: map[string]interface{}{}}
		}
		out = append(out, ns)
	}
	if len(out) == 0 {
		out = append(out, &model.NamespaceInfo{NamespaceID: model.DefaultNamespace, DisplayName: "Public", Visibility: model.ScopePublic, CreateTime: model.NowMillis(), UpdateTime: model.NowMillis(), Metadata: map[string]interface{}{}})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].NamespaceID < out[j].NamespaceID })
	return out, nil
}

func (s *Store) SaveNamespaceMember(m *model.NamespaceMember) error {
	if m == nil {
		return nil
	}
	if m.NamespaceID == "" {
		m.NamespaceID = model.DefaultNamespace
	}
	if m.SubjectID == "" {
		return errors.New("subjectId is required")
	}
	if m.CreateTime == 0 {
		m.CreateTime = model.NowMillis()
	}
	m.UpdateTime = model.NowMillis()
	path := s.namespaceMemberPath(m.NamespaceID, m.SubjectID)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

func (s *Store) DeleteNamespaceMember(namespaceID, subjectID string) error {
	return os.Remove(s.namespaceMemberPath(namespaceID, subjectID))
}

func (s *Store) ListNamespaceMembers(q model.NamespaceMemberQuery) ([]*model.NamespaceMember, int64, error) {
	dir := filepath.Join(s.root, safe(q.NamespaceID), "_members")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []*model.NamespaceMember{}, 0, nil
		}
		return nil, 0, err
	}
	out := []*model.NamespaceMember{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, 0, err
		}
		var m model.NamespaceMember
		if err := json.Unmarshal(b, &m); err != nil {
			return nil, 0, err
		}
		if q.SubjectID != "" && m.SubjectID != q.SubjectID {
			continue
		}
		out = append(out, &m)
	}
	return out, int64(len(out)), nil
}

func writeJSONFile(path string, v interface{}) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}
func readJSONFile(path string, v interface{}) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}

func (s *Store) SetStar(namespaceID, skillName, subjectID string, starred bool) error {
	path := filepath.Join(s.socialDir(namespaceID, skillName), "stars", safe(subjectID)+".json")
	if starred {
		return writeJSONFile(path, map[string]interface{}{"subjectId": subjectID, "time": model.NowMillis()})
	}
	return os.Remove(path)
}
func (s *Store) SetRating(r *model.RatingRecord) error {
	if r == nil {
		return nil
	}
	if r.Rating < 1 || r.Rating > 5 {
		return errors.New("rating must be 1-5")
	}
	if r.CreateTime == 0 {
		r.CreateTime = model.NowMillis()
	}
	r.UpdateTime = model.NowMillis()
	return writeJSONFile(filepath.Join(s.socialDir(r.NamespaceID, r.SkillName), "ratings", safe(r.SubjectID)+".json"), r)
}
func (s *Store) SetSubscription(namespaceID, targetType, targetName, subjectID string, subscribed bool) error {
	path := filepath.Join(s.root, safe(namespaceID), "_subscriptions", safe(targetType), safe(targetName), safe(subjectID)+".json")
	if subscribed {
		return writeJSONFile(path, model.SubscriptionRecord{NamespaceID: namespaceID, TargetType: targetType, TargetName: targetName, SubjectID: subjectID, CreateTime: model.NowMillis()})
	}
	return os.Remove(path)
}
func countFiles(dir string) int64 {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	var n int64
	for _, e := range entries {
		if !e.IsDir() {
			n++
		}
	}
	return n
}
func (s *Store) ListSubscribers(namespaceID, targetType, targetName string) ([]string, error) {
	dir := filepath.Join(s.root, safe(namespaceID), "_subscriptions", safe(targetType), safe(targetName))
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []string{}, nil
		}
		return nil, err
	}
	out := []string{}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		out = append(out, strings.TrimSuffix(e.Name(), filepath.Ext(e.Name())))
	}
	return out, nil
}

func (s *Store) GetSkillSocialStats(namespaceID, skillName, subjectID string) (*model.SkillSocialStats, error) {
	base := s.socialDir(namespaceID, skillName)
	st := &model.SkillSocialStats{NamespaceID: namespaceID, SkillName: skillName}
	st.Stars = countFiles(filepath.Join(base, "stars"))
	st.Subscribers = countFiles(filepath.Join(s.root, safe(namespaceID), "_subscriptions", safe(model.SubscriptionTargetSkill), safe(skillName)))
	if subjectID != "" {
		if _, err := os.Stat(filepath.Join(base, "stars", safe(subjectID)+".json")); err == nil {
			st.MyStarred = true
		}
		if _, err := os.Stat(filepath.Join(s.root, safe(namespaceID), "_subscriptions", safe(model.SubscriptionTargetSkill), safe(skillName), safe(subjectID)+".json")); err == nil {
			st.MySubscribed = true
		}
	}
	rdir := filepath.Join(base, "ratings")
	entries, _ := os.ReadDir(rdir)
	var sum int64
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		var r model.RatingRecord
		if readJSONFile(filepath.Join(rdir, e.Name()), &r) == nil {
			st.RatingCount++
			sum += int64(r.Rating)
			if subjectID != "" && r.SubjectID == subjectID {
				st.MyRating = r.Rating
			}
		}
	}
	if st.RatingCount > 0 {
		st.RatingAverage = float64(sum) / float64(st.RatingCount)
	}
	return st, nil
}

func (s *Store) AppendAudit(l *model.AuditLog) error {
	if l == nil {
		return nil
	}
	if l.ID == "" {
		l.ID = strconv.FormatInt(model.NowMillis(), 10) + "-" + safe(l.Action)
	}
	if l.CreateTime == 0 {
		l.CreateTime = model.NowMillis()
	}
	return writeJSONFile(filepath.Join(s.auditDir(), safe(l.ID)+".json"), l)
}
func (s *Store) ListAuditLogs(q model.AuditQuery) ([]*model.AuditLog, int64, error) {
	entries, err := os.ReadDir(s.auditDir())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []*model.AuditLog{}, 0, nil
		}
		return nil, 0, err
	}
	out := []*model.AuditLog{}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		var l model.AuditLog
		if readJSONFile(filepath.Join(s.auditDir(), e.Name()), &l) != nil {
			continue
		}
		if q.NamespaceID != "" && l.NamespaceID != q.NamespaceID {
			continue
		}
		if q.ResourceType != "" && l.ResourceType != q.ResourceType {
			continue
		}
		if q.ResourceName != "" && l.ResourceName != q.ResourceName {
			continue
		}
		if q.Action != "" && l.Action != q.Action {
			continue
		}
		if q.Operator != "" && l.Operator != q.Operator {
			continue
		}
		out = append(out, &l)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreateTime > out[j].CreateTime })
	return out, int64(len(out)), nil
}

func (s *Store) SaveToken(t *model.TokenInfo) error {
	if t == nil {
		return nil
	}
	if t.CreateTime == 0 {
		t.CreateTime = model.NowMillis()
	}
	if t.Status == "" {
		t.Status = "active"
	}
	return writeJSONFile(s.tokenPath(t.KeyID), t)
}
func (s *Store) DeleteToken(keyID string) error { return os.Remove(s.tokenPath(keyID)) }
func (s *Store) ListTokens(subjectID string) ([]*model.TokenInfo, error) {
	dir := filepath.Join(s.root, "_tokens")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []*model.TokenInfo{}, nil
		}
		return nil, err
	}
	out := []*model.TokenInfo{}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		var t model.TokenInfo
		if readJSONFile(filepath.Join(dir, e.Name()), &t) != nil {
			continue
		}
		if subjectID != "" && t.SubjectID != subjectID {
			continue
		}
		t.Token = ""
		out = append(out, &t)
	}
	return out, nil
}

func (s *Store) FindActiveTokenByHash(tokenHash string) (*model.TokenInfo, error) {
	if tokenHash == "" {
		return nil, nil
	}
	dir := filepath.Join(s.root, "_tokens")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		var t model.TokenInfo
		if readJSONFile(filepath.Join(dir, e.Name()), &t) != nil {
			continue
		}
		if t.Status != "active" || t.TokenHash != tokenHash {
			continue
		}
		if t.ExpiresAt > 0 && t.ExpiresAt < model.NowMillis() {
			continue
		}
		t.Token = ""
		return &t, nil
	}
	return nil, nil
}

func (s *Store) notificationPath(id string) string {
	return filepath.Join(s.root, "_notifications", id+".json")
}
func (s *Store) AppendNotification(n *model.Notification) error {
	if n == nil {
		return nil
	}
	return writeJSONFile(s.notificationPath(n.ID), n)
}
func (s *Store) ListNotifications(q model.NotificationQuery) ([]*model.Notification, int64, error) {
	dir := filepath.Join(s.root, "_notifications")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []*model.Notification{}, 0, nil
		}
		return nil, 0, err
	}
	out := []*model.Notification{}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		var n model.Notification
		if readJSONFile(filepath.Join(dir, e.Name()), &n) != nil {
			continue
		}
		if q.SubjectID != "" && n.SubjectID != q.SubjectID {
			continue
		}
		if q.UnreadOnly && n.Read {
			continue
		}
		out = append(out, &n)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreateTime > out[j].CreateTime })
	return out, int64(len(out)), nil
}
func (s *Store) MarkNotificationRead(subjectID, notificationID string) error {
	p := s.notificationPath(notificationID)
	var n model.Notification
	if err := readJSONFile(p, &n); err != nil {
		return err
	}
	if subjectID != "" && n.SubjectID != subjectID {
		return nil
	}
	n.Read = true
	return writeJSONFile(p, &n)
}

func (s *Store) SaveIdempotency(r *model.IdempotencyRecord) error {
	if r == nil {
		return nil
	}
	return writeJSONFile(s.idempotencyPath(r.Key), r)
}
func (s *Store) LoadIdempotency(key string) (*model.IdempotencyRecord, error) {
	var r model.IdempotencyRecord
	if err := readJSONFile(s.idempotencyPath(key), &r); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	if r.ExpiresAt > 0 && r.ExpiresAt < model.NowMillis() {
		_ = os.Remove(s.idempotencyPath(key))
		return nil, nil
	}
	return &r, nil
}

func (s *Store) catalogEventsPath() string {
	return filepath.Join(s.root, "_catalog_events", "events.json")
}

func (s *Store) AppendCatalogEvent(e *model.CatalogEvent) error {
	if e == nil {
		return nil
	}
	if e.App == "" {
		e.App = model.DefaultApp
	}
	if e.CreatedAt == 0 {
		e.CreatedAt = model.NowMillis()
	}
	if e.ID == 0 {
		e.ID = time.Now().UnixNano()
	}
	path := s.catalogEventsPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	var events []*model.CatalogEvent
	if b, err := os.ReadFile(path); err == nil && len(b) > 0 {
		_ = json.Unmarshal(b, &events)
	}
	events = append(events, e)
	if len(events) > 5000 {
		events = events[len(events)-5000:]
	}
	b, err := json.MarshalIndent(events, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(b, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func (s *Store) ListCatalogEvents(q model.CatalogEventQuery) ([]*model.CatalogEvent, int64, error) {
	path := s.catalogEventsPath()
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []*model.CatalogEvent{}, 0, nil
		}
		return nil, 0, err
	}
	var all []*model.CatalogEvent
	if len(b) > 0 {
		if err := json.Unmarshal(b, &all); err != nil {
			return nil, 0, err
		}
	}
	out := make([]*model.CatalogEvent, 0, len(all))
	for _, e := range all {
		if e == nil {
			continue
		}
		if q.App != "" && e.App != q.App {
			continue
		}
		if q.SkillSetName != "" && e.SkillSetName != "" && e.SkillSetName != q.SkillSetName {
			continue
		}
		if q.ResourceType != "" && e.ResourceType != q.ResourceType {
			continue
		}
		if q.ResourceID != "" && e.ResourceID != q.ResourceID {
			continue
		}
		if q.SinceID > 0 && e.ID <= q.SinceID {
			continue
		}
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	total := int64(len(out))
	if q.Limit <= 0 || q.Limit > 500 {
		q.Limit = 100
	}
	if len(out) > q.Limit {
		out = out[:q.Limit]
	}
	return out, total, nil
}

func (s *Store) LoadModelProfile(namespaceID, id string) (*model.ModelProfile, error) {
	b, err := os.ReadFile(s.modelProfilePath(namespaceID, id))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var rec model.ModelProfile
	if err := json.Unmarshal(b, &rec); err != nil {
		return nil, err
	}
	if rec.NamespaceID == "" {
		rec.NamespaceID = namespaceID
	}
	if rec.ID == "" {
		rec.ID = id
	}
	return &rec, nil
}
func (s *Store) SaveModelProfile(rec *model.ModelProfile) error {
	if rec == nil {
		return nil
	}
	path := s.modelProfilePath(rec.NamespaceID, rec.ID)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
func (s *Store) DeleteModelProfile(namespaceID, id string) error {
	return os.RemoveAll(filepath.Dir(s.modelProfilePath(namespaceID, id)))
}
func (s *Store) ListModelProfiles(namespaceID string) ([]*model.ModelProfile, error) {
	dir := filepath.Join(s.root, "model-profiles")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []*model.ModelProfile{}, nil
		}
		return nil, err
	}
	out := make([]*model.ModelProfile, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		rec, err := s.LoadModelProfile(namespaceID, unsafename(e.Name()))
		if err != nil {
			return nil, err
		}
		if rec != nil {
			out = append(out, rec)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UpdateTime > out[j].UpdateTime })
	return out, nil
}
func (s *Store) modelProfilePath(namespaceID, id string) string {
	return filepath.Join(s.root, "model-profiles", safe(id), "meta.json")
}
