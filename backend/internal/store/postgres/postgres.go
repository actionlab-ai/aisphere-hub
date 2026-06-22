package postgresstore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/actionlab-ai/aisphere-hub/backend/internal/model"
	"github.com/actionlab-ai/aisphere-hub/backend/internal/store"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const systemNamespace = "_system"

type Store struct{ pool *pgxpool.Pool }

func New(dsn string) (*Store, error) {
	return NewWithAutoCreate(dsn, true)
}

func NewWithAutoCreate(dsn string, autoCreate bool) (*Store, error) {
	if strings.TrimSpace(dsn) == "" {
		return nil, fmt.Errorf("postgres dsn is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	pcfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	if autoCreate {
		if err := ensureDatabase(ctx, pcfg.ConnConfig); err != nil {
			return nil, err
		}
	}
	if pcfg.MaxConns == 0 {
		pcfg.MaxConns = 50
	}
	pcfg.MaxConnLifetime = 30 * time.Minute
	pool, err := pgxpool.NewWithConfig(ctx, pcfg)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	s := &Store{pool: pool}
	if autoCreate {
		if err := s.AutoMigrate(ctx); err != nil {
			pool.Close()
			return nil, err
		}
	}
	return s, nil
}
func (s *Store) Close() error                    { s.pool.Close(); return nil }
func (s *Store) WithWrite(fn func() error) error { return fn() }
func (s *Store) WithRead(fn func() error) error  { return fn() }

func (s *Store) AutoMigrate(ctx context.Context) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS aihub_document (
          namespace_id TEXT NOT NULL,
          kind TEXT NOT NULL,
          id TEXT NOT NULL,
          payload JSONB NOT NULL,
          created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
          updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
          PRIMARY KEY(namespace_id, kind, id)
        )`,
		`CREATE INDEX IF NOT EXISTS idx_aihub_document_kind_updated ON aihub_document(kind, updated_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_aihub_document_payload_gin ON aihub_document USING GIN(payload)`,
		`CREATE TABLE IF NOT EXISTS aihub_sequence (name TEXT PRIMARY KEY, value BIGINT NOT NULL DEFAULT 0)`,
	}
	for _, stmt := range stmts {
		if _, err := s.pool.Exec(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}

func normalizeNS(ns string) string {
	if strings.TrimSpace(ns) == "" {
		return model.DefaultNamespace
	}
	return strings.TrimSpace(ns)
}
func nowMillis() int64 { return time.Now().UnixMilli() }
func page(pageNo, pageSize int) (int, int) {
	if pageSize <= 0 || pageSize > 500 {
		pageSize = 100
	}
	if pageNo <= 0 {
		pageNo = 1
	}
	return (pageNo - 1) * pageSize, pageSize
}

func (s *Store) saveDoc(ctx context.Context, ns, kind, id string, v any) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("%s id is required", kind)
	}
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `INSERT INTO aihub_document(namespace_id,kind,id,payload,created_at,updated_at) VALUES($1,$2,$3,$4::jsonb,now(),now()) ON CONFLICT(namespace_id,kind,id) DO UPDATE SET payload=EXCLUDED.payload,updated_at=now()`, normalizeNS(ns), kind, id, string(b))
	return err
}
func (s *Store) loadDoc(ctx context.Context, ns, kind, id string, out any) (bool, error) {
	var b []byte
	err := s.pool.QueryRow(ctx, `SELECT payload FROM aihub_document WHERE namespace_id=$1 AND kind=$2 AND id=$3`, normalizeNS(ns), kind, id).Scan(&b)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, json.Unmarshal(b, out)
}
func (s *Store) deleteDoc(ctx context.Context, ns, kind, id string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM aihub_document WHERE namespace_id=$1 AND kind=$2 AND id=$3`, normalizeNS(ns), kind, id)
	return err
}
func listDocs[T any](s *Store, ctx context.Context, ns, kind string) ([]*T, error) {
	rows, err := s.pool.Query(ctx, `SELECT payload FROM aihub_document WHERE namespace_id=$1 AND kind=$2 ORDER BY updated_at DESC`, normalizeNS(ns), kind)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*T{}
	for rows.Next() {
		var b []byte
		if err := rows.Scan(&b); err != nil {
			return nil, err
		}
		var v T
		if err := json.Unmarshal(b, &v); err != nil {
			return nil, err
		}
		out = append(out, &v)
	}
	return out, rows.Err()
}
func listAllDocs[T any](s *Store, ctx context.Context, kind string) ([]*T, error) {
	rows, err := s.pool.Query(ctx, `SELECT payload FROM aihub_document WHERE kind=$1 ORDER BY updated_at DESC`, kind)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*T{}
	for rows.Next() {
		var b []byte
		if err := rows.Scan(&b); err != nil {
			return nil, err
		}
		var v T
		if err := json.Unmarshal(b, &v); err != nil {
			return nil, err
		}
		out = append(out, &v)
	}
	return out, rows.Err()
}

func (s *Store) Load(namespaceID, name string) (*model.SkillRecord, error) {
	var v model.SkillRecord
	ok, err := s.loadDoc(context.Background(), namespaceID, "skill", name, &v)
	if !ok || err != nil {
		return nil, err
	}
	return &v, nil
}
func (s *Store) Save(rec *model.SkillRecord) error {
	if rec == nil {
		return nil
	}
	if rec.NamespaceID == "" {
		rec.NamespaceID = model.DefaultNamespace
	}
	if rec.CreateTime == 0 {
		rec.CreateTime = nowMillis()
	}
	rec.UpdateTime = nowMillis()
	return s.saveDoc(context.Background(), rec.NamespaceID, "skill", rec.Name, rec)
}
func (s *Store) Delete(namespaceID, name string) error {
	return s.deleteDoc(context.Background(), namespaceID, "skill", name)
}
func (s *Store) List(namespaceID string) ([]*model.SkillRecord, error) {
	return listDocs[model.SkillRecord](s, context.Background(), namespaceID, "skill")
}

func (s *Store) LoadAgent(ns, id string) (*model.AgentRecord, error) {
	var v model.AgentRecord
	ok, err := s.loadDoc(context.Background(), ns, "agent", id, &v)
	if !ok || err != nil {
		return nil, err
	}
	return &v, nil
}
func (s *Store) SaveAgent(rec *model.AgentRecord) error {
	if rec == nil {
		return nil
	}
	if rec.NamespaceID == "" {
		rec.NamespaceID = model.DefaultNamespace
	}
	if rec.CreateTime == 0 {
		rec.CreateTime = nowMillis()
	}
	rec.UpdateTime = nowMillis()
	return s.saveDoc(context.Background(), rec.NamespaceID, "agent", rec.ID, rec)
}
func (s *Store) DeleteAgent(ns, id string) error {
	return s.deleteDoc(context.Background(), ns, "agent", id)
}
func (s *Store) ListAgents(ns string) ([]*model.AgentRecord, error) {
	return listDocs[model.AgentRecord](s, context.Background(), ns, "agent")
}

func (s *Store) LoadTool(ns, id string) (*model.ToolRecord, error) {
	var v model.ToolRecord
	ok, err := s.loadDoc(context.Background(), ns, "tool", id, &v)
	if !ok || err != nil {
		return nil, err
	}
	return &v, nil
}
func (s *Store) SaveTool(rec *model.ToolRecord) error {
	if rec == nil {
		return nil
	}
	if rec.NamespaceID == "" {
		rec.NamespaceID = model.DefaultNamespace
	}
	if rec.CreateTime == 0 {
		rec.CreateTime = nowMillis()
	}
	rec.UpdateTime = nowMillis()
	return s.saveDoc(context.Background(), rec.NamespaceID, "tool", rec.ID, rec)
}
func (s *Store) DeleteTool(ns, id string) error {
	return s.deleteDoc(context.Background(), ns, "tool", id)
}
func (s *Store) ListTools(ns string) ([]*model.ToolRecord, error) {
	return listDocs[model.ToolRecord](s, context.Background(), ns, "tool")
}

func (s *Store) AppendToolFailure(f *model.ToolFailureRecord) error {
	if f == nil {
		return nil
	}
	if f.ID == "" {
		f.ID = fmt.Sprintf("failure_%d", time.Now().UnixNano())
	}
	if f.CreateTime == 0 {
		f.CreateTime = nowMillis()
	}
	ns := f.NamespaceID
	if ns == "" {
		ns = model.DefaultNamespace
	}
	return s.saveDoc(context.Background(), ns, "tool_failure", f.ID, f)
}
func (s *Store) ListToolFailures(q model.ToolFailureQuery) ([]*model.ToolFailureRecord, int64, error) {
	items, err := listAllDocs[model.ToolFailureRecord](s, context.Background(), "tool_failure")
	if err != nil {
		return nil, 0, err
	}
	out := []*model.ToolFailureRecord{}
	for _, f := range items {
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
		out = append(out, f)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreateTime > out[j].CreateTime })
	total := int64(len(out))
	lim := q.Limit
	if lim <= 0 || lim > 500 {
		lim = 100
	}
	if len(out) > lim {
		out = out[:lim]
	}
	return out, total, nil
}

func (s *Store) LoadGroup(ns, name string) (*model.SkillGroup, error) {
	var v model.SkillGroup
	ok, err := s.loadDoc(context.Background(), ns, "skillset", name, &v)
	if !ok || err != nil {
		return nil, err
	}
	return &v, nil
}
func (s *Store) SaveGroup(g *model.SkillGroup) error {
	if g == nil {
		return nil
	}
	if g.NamespaceID == "" {
		g.NamespaceID = model.DefaultNamespace
	}
	if g.CreateTime == 0 {
		g.CreateTime = nowMillis()
	}
	g.UpdateTime = nowMillis()
	return s.saveDoc(context.Background(), g.NamespaceID, "skillset", g.Name, g)
}
func (s *Store) DeleteGroup(ns, name string) error {
	return s.deleteDoc(context.Background(), ns, "skillset", name)
}
func (s *Store) ListGroups(ns string) ([]*model.SkillGroup, error) {
	return listDocs[model.SkillGroup](s, context.Background(), ns, "skillset")
}

func (s *Store) SaveProposal(p *model.SkillProposal) error {
	if p == nil {
		return nil
	}
	if p.CreateTime == 0 {
		p.CreateTime = nowMillis()
	}
	p.UpdateTime = nowMillis()
	return s.saveDoc(context.Background(), p.NamespaceID, "proposal", p.ProposalID, p)
}
func (s *Store) LoadProposal(id string) (*model.SkillProposal, error) {
	var v model.SkillProposal
	ok, err := s.loadDoc(context.Background(), model.DefaultNamespace, "proposal", id, &v)
	if ok || err != nil {
		return &v, err
	}
	items, err := listAllDocs[model.SkillProposal](s, context.Background(), "proposal")
	if err != nil {
		return nil, err
	}
	for _, p := range items {
		if p.ProposalID == id {
			return p, nil
		}
	}
	return nil, nil
}
func (s *Store) ListProposals(q model.ProposalQuery) ([]*model.SkillProposal, int64, error) {
	items, err := listAllDocs[model.SkillProposal](s, context.Background(), "proposal")
	if err != nil {
		return nil, 0, err
	}
	out := []*model.SkillProposal{}
	for _, p := range items {
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
	total := int64(len(out))
	off, lim := page(q.PageNo, q.PageSize)
	if off > len(out) {
		return []*model.SkillProposal{}, total, nil
	}
	end := off + lim
	if end > len(out) {
		end = len(out)
	}
	return out[off:end], total, nil
}
func (s *Store) SaveOverlay(o *model.SkillOverlay) error {
	if o == nil {
		return nil
	}
	if o.CreateTime == 0 {
		o.CreateTime = nowMillis()
	}
	return s.saveDoc(context.Background(), o.NamespaceID, "overlay", o.OverlayRef, o)
}
func (s *Store) LoadOverlay(ref string) (*model.SkillOverlay, error) {
	items, err := listAllDocs[model.SkillOverlay](s, context.Background(), "overlay")
	if err != nil {
		return nil, err
	}
	for _, o := range items {
		if o.OverlayRef == ref {
			return o, nil
		}
	}
	return nil, nil
}
func (s *Store) SaveProposalValidation(v *model.ProposalValidation) error {
	if v == nil {
		return nil
	}
	if v.CreateTime == 0 {
		v.CreateTime = nowMillis()
	}
	id := fmt.Sprintf("%s:%d", v.ProposalID, v.CreateTime)
	return s.saveDoc(context.Background(), model.DefaultNamespace, "proposal_validation", id, v)
}
func (s *Store) ListProposalValidations(proposalID string) ([]*model.ProposalValidation, error) {
	items, err := listAllDocs[model.ProposalValidation](s, context.Background(), "proposal_validation")
	if err != nil {
		return nil, err
	}
	out := []*model.ProposalValidation{}
	for _, v := range items {
		if v.ProposalID == proposalID {
			out = append(out, v)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreateTime > out[j].CreateTime })
	return out, nil
}

func (s *Store) SaveNamespace(ns *model.NamespaceInfo) error {
	if ns == nil {
		return nil
	}
	if ns.CreateTime == 0 {
		ns.CreateTime = nowMillis()
	}
	ns.UpdateTime = nowMillis()
	return s.saveDoc(context.Background(), systemNamespace, "namespace", ns.NamespaceID, ns)
}
func (s *Store) LoadNamespace(namespaceID string) (*model.NamespaceInfo, error) {
	var v model.NamespaceInfo
	ok, err := s.loadDoc(context.Background(), systemNamespace, "namespace", namespaceID, &v)
	if !ok || err != nil {
		return nil, err
	}
	return &v, nil
}
func (s *Store) ListNamespaces() ([]*model.NamespaceInfo, error) {
	return listDocs[model.NamespaceInfo](s, context.Background(), systemNamespace, "namespace")
}
func (s *Store) SaveNamespaceMember(m *model.NamespaceMember) error {
	if m == nil {
		return nil
	}
	if m.CreateTime == 0 {
		m.CreateTime = nowMillis()
	}
	m.UpdateTime = nowMillis()
	return s.saveDoc(context.Background(), m.NamespaceID, "namespace_member", m.SubjectID, m)
}
func (s *Store) DeleteNamespaceMember(ns, subjectID string) error {
	return s.deleteDoc(context.Background(), ns, "namespace_member", subjectID)
}
func (s *Store) ListNamespaceMembers(q model.NamespaceMemberQuery) ([]*model.NamespaceMember, int64, error) {
	items, err := listDocs[model.NamespaceMember](s, context.Background(), q.NamespaceID, "namespace_member")
	if err != nil {
		return nil, 0, err
	}
	out := []*model.NamespaceMember{}
	for _, m := range items {
		if q.SubjectID != "" && m.SubjectID != q.SubjectID {
			continue
		}
		out = append(out, m)
	}
	total := int64(len(out))
	off, lim := page(q.PageNo, q.PageSize)
	if off > len(out) {
		return []*model.NamespaceMember{}, total, nil
	}
	end := off + lim
	if end > len(out) {
		end = len(out)
	}
	return out[off:end], total, nil
}

func (s *Store) SetStar(ns, skillName, subjectID string, starred bool) error {
	id := skillName + ":" + subjectID
	if !starred {
		return s.deleteDoc(context.Background(), ns, "star", id)
	}
	return s.saveDoc(context.Background(), ns, "star", id, map[string]string{"skillName": skillName, "subjectId": subjectID})
}
func (s *Store) SetRating(r *model.RatingRecord) error {
	if r == nil {
		return nil
	}
	if r.CreateTime == 0 {
		r.CreateTime = nowMillis()
	}
	r.UpdateTime = nowMillis()
	return s.saveDoc(context.Background(), r.NamespaceID, "rating", r.SkillName+":"+r.SubjectID, r)
}
func (s *Store) SetSubscription(ns, targetType, targetName, subjectID string, subscribed bool) error {
	id := targetType + ":" + targetName + ":" + subjectID
	if !subscribed {
		return s.deleteDoc(context.Background(), ns, "subscription", id)
	}
	rec := model.SubscriptionRecord{NamespaceID: ns, TargetType: targetType, TargetName: targetName, SubjectID: subjectID, CreateTime: nowMillis()}
	return s.saveDoc(context.Background(), ns, "subscription", id, &rec)
}
func (s *Store) ListSubscribers(ns, targetType, targetName string) ([]string, error) {
	items, err := listDocs[model.SubscriptionRecord](s, context.Background(), ns, "subscription")
	if err != nil {
		return nil, err
	}
	out := []string{}
	for _, r := range items {
		if r.TargetType == targetType && r.TargetName == targetName {
			out = append(out, r.SubjectID)
		}
	}
	return out, nil
}
func (s *Store) GetSkillSocialStats(ns, skillName, subjectID string) (*model.SkillSocialStats, error) {
	stats := &model.SkillSocialStats{NamespaceID: ns, SkillName: skillName}
	stars, _ := listDocs[map[string]string](s, context.Background(), ns, "star")
	for _, st := range stars {
		if (*st)["skillName"] == skillName {
			stats.Stars++
			if (*st)["subjectId"] == subjectID {
				stats.MyStarred = true
			}
		}
	}
	ratings, _ := listDocs[model.RatingRecord](s, context.Background(), ns, "rating")
	var sum int
	for _, r := range ratings {
		if r.SkillName == skillName {
			stats.RatingCount++
			sum += r.Rating
			if r.SubjectID == subjectID {
				stats.MyRating = r.Rating
			}
		}
	}
	if stats.RatingCount > 0 {
		stats.RatingAverage = float64(sum) / float64(stats.RatingCount)
	}
	subs, _ := listDocs[model.SubscriptionRecord](s, context.Background(), ns, "subscription")
	for _, sub := range subs {
		if sub.TargetType == model.SubscriptionTargetSkill && sub.TargetName == skillName {
			stats.Subscribers++
			if sub.SubjectID == subjectID {
				stats.MySubscribed = true
			}
		}
	}
	if rec, _ := s.Load(ns, skillName); rec != nil {
		stats.DownloadCount = rec.DownloadCount
	}
	return stats, nil
}

func (s *Store) AppendAudit(l *model.AuditLog) error {
	if l == nil {
		return nil
	}
	if l.ID == "" {
		l.ID = fmt.Sprintf("audit_%d", time.Now().UnixNano())
	}
	if l.CreateTime == 0 {
		l.CreateTime = nowMillis()
	}
	return s.saveDoc(context.Background(), l.NamespaceID, "audit", l.ID, l)
}
func (s *Store) ListAuditLogs(q model.AuditQuery) ([]*model.AuditLog, int64, error) {
	items, err := listAllDocs[model.AuditLog](s, context.Background(), "audit")
	if err != nil {
		return nil, 0, err
	}
	out := []*model.AuditLog{}
	for _, l := range items {
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
		out = append(out, l)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreateTime > out[j].CreateTime })
	total := int64(len(out))
	off, lim := page(q.PageNo, q.PageSize)
	if off > len(out) {
		return []*model.AuditLog{}, total, nil
	}
	end := off + lim
	if end > len(out) {
		end = len(out)
	}
	return out[off:end], total, nil
}

func (s *Store) SaveToken(t *model.TokenInfo) error {
	if t == nil {
		return nil
	}
	if t.CreateTime == 0 {
		t.CreateTime = nowMillis()
	}
	return s.saveDoc(context.Background(), systemNamespace, "token", t.KeyID, t)
}
func (s *Store) DeleteToken(keyID string) error {
	var t model.TokenInfo
	ok, err := s.loadDoc(context.Background(), systemNamespace, "token", keyID, &t)
	if err != nil || !ok {
		return err
	}
	t.Status = "deleted"
	return s.saveDoc(context.Background(), systemNamespace, "token", keyID, &t)
}
func (s *Store) ListTokens(subjectID string) ([]*model.TokenInfo, error) {
	items, err := listDocs[model.TokenInfo](s, context.Background(), systemNamespace, "token")
	if err != nil {
		return nil, err
	}
	out := []*model.TokenInfo{}
	for _, t := range items {
		if t.SubjectID == subjectID && t.Status != "deleted" {
			out = append(out, t)
		}
	}
	return out, nil
}
func (s *Store) FindActiveTokenByHash(hash string) (*model.TokenInfo, error) {
	items, err := listDocs[model.TokenInfo](s, context.Background(), systemNamespace, "token")
	if err != nil {
		return nil, err
	}
	now := nowMillis()
	for _, t := range items {
		if t.TokenHash == hash && t.Status == "active" && (t.ExpiresAt == 0 || t.ExpiresAt > now) {
			return t, nil
		}
	}
	return nil, nil
}
func (s *Store) AppendNotification(n *model.Notification) error {
	if n == nil {
		return nil
	}
	if n.ID == "" {
		n.ID = fmt.Sprintf("notif_%d", time.Now().UnixNano())
	}
	if n.CreateTime == 0 {
		n.CreateTime = nowMillis()
	}
	return s.saveDoc(context.Background(), systemNamespace, "notification", n.SubjectID+":"+n.ID, n)
}
func (s *Store) ListNotifications(q model.NotificationQuery) ([]*model.Notification, int64, error) {
	items, err := listDocs[model.Notification](s, context.Background(), systemNamespace, "notification")
	if err != nil {
		return nil, 0, err
	}
	out := []*model.Notification{}
	for _, n := range items {
		if q.SubjectID != "" && n.SubjectID != q.SubjectID {
			continue
		}
		if q.UnreadOnly && n.Read {
			continue
		}
		out = append(out, n)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreateTime > out[j].CreateTime })
	total := int64(len(out))
	off, lim := page(q.PageNo, q.PageSize)
	if off > len(out) {
		return []*model.Notification{}, total, nil
	}
	end := off + lim
	if end > len(out) {
		end = len(out)
	}
	return out[off:end], total, nil
}
func (s *Store) MarkNotificationRead(subjectID, notificationID string) error {
	var n model.Notification
	id := subjectID + ":" + notificationID
	ok, err := s.loadDoc(context.Background(), systemNamespace, "notification", id, &n)
	if err != nil || !ok {
		return err
	}
	n.Read = true
	return s.saveDoc(context.Background(), systemNamespace, "notification", id, &n)
}
func (s *Store) SaveIdempotency(r *model.IdempotencyRecord) error {
	if r == nil {
		return nil
	}
	if r.CreateTime == 0 {
		r.CreateTime = nowMillis()
	}
	return s.saveDoc(context.Background(), systemNamespace, "idempotency", r.Key, r)
}
func (s *Store) LoadIdempotency(key string) (*model.IdempotencyRecord, error) {
	var v model.IdempotencyRecord
	ok, err := s.loadDoc(context.Background(), systemNamespace, "idempotency", key, &v)
	if !ok || err != nil {
		return nil, err
	}
	if v.ExpiresAt > 0 && v.ExpiresAt < nowMillis() {
		_ = s.deleteDoc(context.Background(), systemNamespace, "idempotency", key)
		return nil, nil
	}
	return &v, nil
}

func (s *Store) nextSeq(ctx context.Context, name string) (int64, error) {
	var v int64
	err := s.pool.QueryRow(ctx, `INSERT INTO aihub_sequence(name,value) VALUES($1,1) ON CONFLICT(name) DO UPDATE SET value=aihub_sequence.value+1 RETURNING value`, name).Scan(&v)
	return v, err
}
func (s *Store) AppendCatalogEvent(e *model.CatalogEvent) error {
	if e == nil {
		return nil
	}
	if e.ID == 0 {
		id, err := s.nextSeq(context.Background(), "catalog_event")
		if err != nil {
			return err
		}
		e.ID = id
	}
	if e.CreatedAt == 0 {
		e.CreatedAt = nowMillis()
	}
	return s.saveDoc(context.Background(), systemNamespace, "catalog_event", fmt.Sprint(e.ID), e)
}
func (s *Store) ListCatalogEvents(q model.CatalogEventQuery) ([]*model.CatalogEvent, int64, error) {
	items, err := listDocs[model.CatalogEvent](s, context.Background(), systemNamespace, "catalog_event")
	if err != nil {
		return nil, 0, err
	}
	out := []*model.CatalogEvent{}
	for _, e := range items {
		if q.App != "" && e.App != q.App {
			continue
		}
		if q.SkillSetName != "" && e.SkillSetName != q.SkillSetName {
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
	lim := q.Limit
	if lim <= 0 || lim > 500 {
		lim = 100
	}
	if len(out) > lim {
		out = out[:lim]
	}
	return out, total, nil
}

func (s *Store) LoadSandboxProfile(ns, id string) (*model.SandboxProfile, error) {
	var v model.SandboxProfile
	ok, err := s.loadDoc(context.Background(), ns, "sandbox_profile", id, &v)
	if !ok || err != nil {
		return nil, err
	}
	return &v, nil
}
func (s *Store) SaveSandboxProfile(p *model.SandboxProfile) error {
	if p == nil {
		return nil
	}
	if p.NamespaceID == "" {
		p.NamespaceID = model.DefaultNamespace
	}
	if p.CreateTime == 0 {
		p.CreateTime = nowMillis()
	}
	p.UpdateTime = nowMillis()
	return s.saveDoc(context.Background(), p.NamespaceID, "sandbox_profile", p.ID, p)
}
func (s *Store) DeleteSandboxProfile(ns, id string) error {
	return s.deleteDoc(context.Background(), ns, "sandbox_profile", id)
}
func (s *Store) ListSandboxProfiles(ns string) ([]*model.SandboxProfile, error) {
	return listDocs[model.SandboxProfile](s, context.Background(), ns, "sandbox_profile")
}
func (s *Store) LoadSandboxPolicy(ns, id string) (*model.SandboxPolicy, error) {
	var v model.SandboxPolicy
	ok, err := s.loadDoc(context.Background(), ns, "sandbox_policy", id, &v)
	if !ok || err != nil {
		return nil, err
	}
	return &v, nil
}
func (s *Store) SaveSandboxPolicy(p *model.SandboxPolicy) error {
	if p == nil {
		return nil
	}
	if p.NamespaceID == "" {
		p.NamespaceID = model.DefaultNamespace
	}
	if p.CreateTime == 0 {
		p.CreateTime = nowMillis()
	}
	p.UpdateTime = nowMillis()
	return s.saveDoc(context.Background(), p.NamespaceID, "sandbox_policy", p.ID, p)
}
func (s *Store) DeleteSandboxPolicy(ns, id string) error {
	return s.deleteDoc(context.Background(), ns, "sandbox_policy", id)
}
func (s *Store) ListSandboxPolicies(ns string) ([]*model.SandboxPolicy, error) {
	return listDocs[model.SandboxPolicy](s, context.Background(), ns, "sandbox_policy")
}

func (s *Store) LoadModelProfile(ns, id string) (*model.ModelProfile, error) {
	var v model.ModelProfile
	ok, err := s.loadDoc(context.Background(), ns, "model_profile", id, &v)
	if !ok || err != nil {
		return nil, err
	}
	return &v, nil
}
func (s *Store) SaveModelProfile(p *model.ModelProfile) error {
	if p == nil {
		return nil
	}
	if p.NamespaceID == "" {
		p.NamespaceID = model.DefaultNamespace
	}
	if p.CreateTime == 0 {
		p.CreateTime = nowMillis()
	}
	p.UpdateTime = nowMillis()
	return s.saveDoc(context.Background(), p.NamespaceID, "model_profile", p.ID, p)
}
func (s *Store) DeleteModelProfile(ns, id string) error {
	return s.deleteDoc(context.Background(), ns, "model_profile", id)
}
func (s *Store) ListModelProfiles(ns string) ([]*model.ModelProfile, error) {
	return listDocs[model.ModelProfile](s, context.Background(), ns, "model_profile")
}

var _ store.Backend = (*Store)(nil)
