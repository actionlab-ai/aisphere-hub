package mysqlstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"sort"
	"time"

	"github.com/actionlab-ai/aisphere-hub/backend/internal/model"
	"github.com/actionlab-ai/aisphere-hub/backend/internal/store"
	_ "github.com/go-sql-driver/mysql"
)

const (
	resourceTypeSkill = "skill"
	resourceTypeAgent = "agent"
	resourceTypeTool  = "tool"
)

type Store struct{ db *sql.DB }

func New(dsn string) (*Store, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(50)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(30 * time.Minute)
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &Store{db: db}, nil
}
func (s *Store) Close() error                    { return s.db.Close() }
func (s *Store) WithWrite(fn func() error) error { return fn() }
func (s *Store) WithRead(fn func() error) error  { return fn() }

func (s *Store) Load(ns, name string) (*model.SkillRecord, error) {
	row := s.db.QueryRowContext(context.Background(), `SELECT app,org_id,project_id,owner_subject,description,status,scope,owner,biz_tags,metadata,download_count,UNIX_TIMESTAMP(created_at)*1000,UNIX_TIMESTAMP(updated_at)*1000 FROM ai_resource WHERE namespace_id=? AND type=? AND name=?`, ns, resourceTypeSkill, name)
	var desc, status, scope, owner string
	var app, orgID, projectID, ownerSubject, biz, meta sql.NullString
	var dl, ct, ut int64
	if err := row.Scan(&app, &orgID, &projectID, &ownerSubject, &desc, &status, &scope, &owner, &biz, &meta, &dl, &ct, &ut); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	rec := &model.SkillRecord{NamespaceID: ns, App: model.DefaultApp, Name: name, Description: desc, Status: status, Scope: scope, Owner: owner, DownloadCount: dl, CreateTime: ct, UpdateTime: ut, Labels: map[string]string{}, Versions: map[string]*model.VersionRecord{}}
	if app.Valid {
		rec.App = app.String
	}
	if orgID.Valid {
		rec.OrgID = orgID.String
	}
	if projectID.Valid {
		rec.ProjectID = projectID.String
	}
	if ownerSubject.Valid {
		rec.OwnerSubject = ownerSubject.String
	}
	if biz.Valid {
		rec.BizTags = biz.String
	}
	if meta.Valid && meta.String != "" {
		_ = json.Unmarshal([]byte(meta.String), rec)
	}
	if rec.NamespaceID == "" {
		rec.NamespaceID = ns
	}
	if rec.Name == "" {
		rec.Name = name
	}
	if rec.Labels == nil {
		rec.Labels = map[string]string{}
	}
	if rec.Versions == nil {
		rec.Versions = map[string]*model.VersionRecord{}
	}
	rows, err := s.db.QueryContext(context.Background(), `SELECT version,status,author,commit_msg,resource_card,storage,publish_pipeline_info,download_count,UNIX_TIMESTAMP(created_at)*1000,UNIX_TIMESTAMP(updated_at)*1000 FROM ai_resource_version WHERE namespace_id=? AND type=? AND name=?`, ns, resourceTypeSkill, name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var v model.VersionRecord
		var author, commit, card, storage, pipeline sql.NullString
		if err := rows.Scan(&v.Version, &v.Status, &author, &commit, &card, &storage, &pipeline, &v.DownloadCount, &v.CreateTime, &v.UpdateTime); err != nil {
			return nil, err
		}
		if author.Valid {
			v.Author = author.String
		}
		if commit.Valid {
			v.CommitMsg = commit.String
		}
		if pipeline.Valid {
			v.PublishPipelineInfo = pipeline.String
		}
		if card.Valid && card.String != "" {
			_ = json.Unmarshal([]byte(card.String), &v.Skill)
		}
		if storage.Valid && storage.String != "" {
			_ = json.Unmarshal([]byte(storage.String), &v)
		}
		if v.Version != "" {
			rec.Versions[v.Version] = &v
		}
	}
	return rec, rows.Err()
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
	meta, _ := json.Marshal(rec)
	biz := rec.BizTags
	if biz == "" {
		biz = "[]"
	}
	now := time.Now()
	if rec.App == "" {
		rec.App = model.DefaultApp
	}
	if rec.OwnerSubject == "" {
		rec.OwnerSubject = rec.Owner
	}
	_, err := s.db.ExecContext(context.Background(), `INSERT INTO ai_resource(namespace_id,type,name,app,org_id,project_id,owner_subject,description,status,scope,owner,biz_tags,metadata,download_count,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?) ON DUPLICATE KEY UPDATE app=VALUES(app),org_id=VALUES(org_id),project_id=VALUES(project_id),owner_subject=VALUES(owner_subject),description=VALUES(description),status=VALUES(status),scope=VALUES(scope),owner=VALUES(owner),biz_tags=VALUES(biz_tags),metadata=VALUES(metadata),download_count=VALUES(download_count),updated_at=VALUES(updated_at)`, rec.NamespaceID, resourceTypeSkill, rec.Name, rec.App, nullable(rec.OrgID), nullable(rec.ProjectID), nullable(rec.OwnerSubject), rec.Description, rec.Status, rec.Scope, rec.Owner, biz, string(meta), rec.DownloadCount, millisTime(rec.CreateTime, now), millisTime(rec.UpdateTime, now))
	if err != nil {
		return err
	}
	for _, v := range rec.Versions {
		card, _ := json.Marshal(v.Skill)
		storage, _ := json.Marshal(v)
		_, err := s.db.ExecContext(context.Background(), `INSERT INTO ai_resource_version(namespace_id,type,name,version,status,author,commit_msg,resource_card,storage,publish_pipeline_info,download_count,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?) ON DUPLICATE KEY UPDATE status=VALUES(status),author=VALUES(author),commit_msg=VALUES(commit_msg),resource_card=VALUES(resource_card),storage=VALUES(storage),publish_pipeline_info=VALUES(publish_pipeline_info),download_count=VALUES(download_count),updated_at=VALUES(updated_at)`, rec.NamespaceID, resourceTypeSkill, rec.Name, v.Version, v.Status, v.Author, v.CommitMsg, string(card), string(storage), nullable(v.PublishPipelineInfo), v.DownloadCount, millisTime(v.CreateTime, now), millisTime(v.UpdateTime, now))
		if err != nil {
			return err
		}
	}
	return nil
}
func (s *Store) Delete(ns, name string) error {
	_, err := s.db.ExecContext(context.Background(), `DELETE FROM ai_resource_version WHERE namespace_id=? AND type=? AND name=?`, ns, resourceTypeSkill, name)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(context.Background(), `DELETE FROM ai_resource WHERE namespace_id=? AND type=? AND name=?`, ns, resourceTypeSkill, name)
	return err
}
func (s *Store) List(ns string) ([]*model.SkillRecord, error) {
	rows, err := s.db.QueryContext(context.Background(), `SELECT name FROM ai_resource WHERE namespace_id=? AND type=? ORDER BY updated_at DESC`, ns, resourceTypeSkill)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*model.SkillRecord{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		rec, err := s.Load(ns, name)
		if err != nil {
			return nil, err
		}
		if rec != nil {
			out = append(out, rec)
		}
	}
	return out, rows.Err()
}

func (s *Store) LoadAgent(ns, id string) (*model.AgentRecord, error) {
	row := s.db.QueryRowContext(context.Background(), `SELECT app,org_id,project_id,owner_subject,description,status,scope,metadata,UNIX_TIMESTAMP(created_at)*1000,UNIX_TIMESTAMP(updated_at)*1000 FROM ai_resource WHERE namespace_id=? AND type=? AND name=?`, ns, resourceTypeAgent, id)
	var app, orgID, projectID, ownerSubject, metadata sql.NullString
	var description, status, scope string
	var createTime, updateTime int64
	if err := row.Scan(&app, &orgID, &projectID, &ownerSubject, &description, &status, &scope, &metadata, &createTime, &updateTime); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	rec := &model.AgentRecord{NamespaceID: ns, App: model.DefaultApp, ID: id, Description: description, Status: status, Scope: scope, CreateTime: createTime, UpdateTime: updateTime, Labels: map[string]string{}, Versions: map[string]*model.AgentVersionRecord{}}
	if app.Valid {
		rec.App = app.String
	}
	if orgID.Valid {
		rec.OrgID = orgID.String
	}
	if projectID.Valid {
		rec.ProjectID = projectID.String
	}
	if ownerSubject.Valid {
		rec.OwnerSubject = ownerSubject.String
	}
	if metadata.Valid && metadata.String != "" {
		_ = json.Unmarshal([]byte(metadata.String), rec)
	}
	if rec.NamespaceID == "" {
		rec.NamespaceID = ns
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
	rows, err := s.db.QueryContext(context.Background(), `SELECT version,storage FROM ai_resource_version WHERE namespace_id=? AND type=? AND name=?`, ns, resourceTypeAgent, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var version string
		var storage sql.NullString
		if err := rows.Scan(&version, &storage); err != nil {
			return nil, err
		}
		entry := &model.AgentVersionRecord{Version: version}
		if storage.Valid && storage.String != "" {
			_ = json.Unmarshal([]byte(storage.String), entry)
		}
		if entry.Version == "" {
			entry.Version = version
		}
		rec.Versions[version] = entry
	}
	return rec, rows.Err()
}

func (s *Store) SaveAgent(rec *model.AgentRecord) error {
	if rec == nil {
		return nil
	}
	if rec.App == "" {
		rec.App = model.DefaultApp
	}
	if rec.Status == "" {
		rec.Status = model.MetaStatusEnable
	}
	if rec.Labels == nil {
		rec.Labels = map[string]string{}
	}
	if rec.Versions == nil {
		rec.Versions = map[string]*model.AgentVersionRecord{}
	}
	meta, _ := json.Marshal(rec)
	now := time.Now()
	_, err := s.db.ExecContext(context.Background(), `INSERT INTO ai_resource(namespace_id,type,name,app,org_id,project_id,owner_subject,description,status,scope,owner,biz_tags,metadata,download_count,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?) ON DUPLICATE KEY UPDATE app=VALUES(app),org_id=VALUES(org_id),project_id=VALUES(project_id),owner_subject=VALUES(owner_subject),description=VALUES(description),status=VALUES(status),scope=VALUES(scope),metadata=VALUES(metadata),updated_at=VALUES(updated_at)`, rec.NamespaceID, resourceTypeAgent, rec.ID, rec.App, nullable(rec.OrgID), nullable(rec.ProjectID), nullable(rec.OwnerSubject), rec.Description, rec.Status, rec.Scope, rec.OwnerSubject, "[]", string(meta), 0, millisTime(rec.CreateTime, now), millisTime(rec.UpdateTime, now))
	if err != nil {
		return err
	}
	for _, version := range rec.Versions {
		if version == nil {
			continue
		}
		definition, _ := json.Marshal(version.Definition)
		storage, _ := json.Marshal(version)
		_, err = s.db.ExecContext(context.Background(), `INSERT INTO ai_resource_version(namespace_id,type,name,version,status,author,commit_msg,resource_card,storage,publish_pipeline_info,download_count,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?) ON DUPLICATE KEY UPDATE status=VALUES(status),author=VALUES(author),commit_msg=VALUES(commit_msg),resource_card=VALUES(resource_card),storage=VALUES(storage),updated_at=VALUES(updated_at)`, rec.NamespaceID, resourceTypeAgent, rec.ID, version.Version, model.VersionStatusOnline, version.Author, version.CommitMsg, string(definition), string(storage), nil, 0, millisTime(version.CreateTime, now), millisTime(rec.UpdateTime, now))
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) DeleteAgent(ns, id string) error {
	if _, err := s.db.ExecContext(context.Background(), `DELETE FROM ai_resource_version WHERE namespace_id=? AND type=? AND name=?`, ns, resourceTypeAgent, id); err != nil {
		return err
	}
	_, err := s.db.ExecContext(context.Background(), `DELETE FROM ai_resource WHERE namespace_id=? AND type=? AND name=?`, ns, resourceTypeAgent, id)
	return err
}

func (s *Store) ListAgents(ns string) ([]*model.AgentRecord, error) {
	rows, err := s.db.QueryContext(context.Background(), `SELECT name FROM ai_resource WHERE namespace_id=? AND type=? ORDER BY updated_at DESC`, ns, resourceTypeAgent)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*model.AgentRecord{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		rec, err := s.LoadAgent(ns, id)
		if err != nil {
			return nil, err
		}
		if rec != nil {
			out = append(out, rec)
		}
	}
	return out, rows.Err()
}
func (s *Store) LoadTool(ns, id string) (*model.ToolRecord, error) {
	row := s.db.QueryRowContext(context.Background(), `SELECT app,org_id,project_id,owner_subject,description,status,scope,metadata,UNIX_TIMESTAMP(created_at)*1000,UNIX_TIMESTAMP(updated_at)*1000 FROM ai_resource WHERE namespace_id=? AND type=? AND name=?`, ns, resourceTypeTool, id)
	var app, orgID, projectID, ownerSubject, metadata sql.NullString
	var description, status, scope string
	var createTime, updateTime int64
	if err := row.Scan(&app, &orgID, &projectID, &ownerSubject, &description, &status, &scope, &metadata, &createTime, &updateTime); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	rec := &model.ToolRecord{NamespaceID: ns, App: model.DefaultApp, ID: id, Description: description, Status: status, Scope: scope, CreateTime: createTime, UpdateTime: updateTime, Labels: map[string]string{}, Versions: map[string]*model.ToolVersionRecord{}}
	if app.Valid {
		rec.App = app.String
	}
	if orgID.Valid {
		rec.OrgID = orgID.String
	}
	if projectID.Valid {
		rec.ProjectID = projectID.String
	}
	if ownerSubject.Valid {
		rec.OwnerSubject = ownerSubject.String
	}
	if metadata.Valid && metadata.String != "" {
		_ = json.Unmarshal([]byte(metadata.String), rec)
	}
	if rec.NamespaceID == "" {
		rec.NamespaceID = ns
	}
	if rec.ID == "" {
		rec.ID = id
	}
	if rec.App == "" {
		rec.App = model.DefaultApp
	}
	if rec.Labels == nil {
		rec.Labels = map[string]string{}
	}
	if rec.Versions == nil {
		rec.Versions = map[string]*model.ToolVersionRecord{}
	}
	rows, err := s.db.QueryContext(context.Background(), `SELECT version,storage FROM ai_resource_version WHERE namespace_id=? AND type=? AND name=?`, ns, resourceTypeTool, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var version string
		var storage sql.NullString
		if err := rows.Scan(&version, &storage); err != nil {
			return nil, err
		}
		entry := &model.ToolVersionRecord{Version: version}
		if storage.Valid && storage.String != "" {
			_ = json.Unmarshal([]byte(storage.String), entry)
		}
		if entry.Version == "" {
			entry.Version = version
		}
		rec.Versions[version] = entry
	}
	return rec, rows.Err()
}

func (s *Store) SaveTool(rec *model.ToolRecord) error {
	if rec == nil {
		return nil
	}
	if rec.App == "" {
		rec.App = model.DefaultApp
	}
	if rec.Status == "" {
		rec.Status = model.MetaStatusEnable
	}
	if rec.Labels == nil {
		rec.Labels = map[string]string{}
	}
	if rec.Versions == nil {
		rec.Versions = map[string]*model.ToolVersionRecord{}
	}
	meta, _ := json.Marshal(rec)
	now := time.Now()
	_, err := s.db.ExecContext(context.Background(), `INSERT INTO ai_resource(namespace_id,type,name,app,org_id,project_id,owner_subject,description,status,scope,owner,biz_tags,metadata,download_count,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?) ON DUPLICATE KEY UPDATE app=VALUES(app),org_id=VALUES(org_id),project_id=VALUES(project_id),owner_subject=VALUES(owner_subject),description=VALUES(description),status=VALUES(status),scope=VALUES(scope),metadata=VALUES(metadata),updated_at=VALUES(updated_at)`, rec.NamespaceID, resourceTypeTool, rec.ID, rec.App, nullable(rec.OrgID), nullable(rec.ProjectID), nullable(rec.OwnerSubject), rec.Description, rec.Status, rec.Scope, rec.OwnerSubject, "[]", string(meta), 0, millisTime(rec.CreateTime, now), millisTime(rec.UpdateTime, now))
	if err != nil {
		return err
	}
	for _, version := range rec.Versions {
		if version == nil {
			continue
		}
		definition, _ := json.Marshal(version.Definition)
		storage, _ := json.Marshal(version)
		_, err = s.db.ExecContext(context.Background(), `INSERT INTO ai_resource_version(namespace_id,type,name,version,status,author,commit_msg,resource_card,storage,publish_pipeline_info,download_count,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?) ON DUPLICATE KEY UPDATE status=VALUES(status),author=VALUES(author),commit_msg=VALUES(commit_msg),resource_card=VALUES(resource_card),storage=VALUES(storage),updated_at=VALUES(updated_at)`, rec.NamespaceID, resourceTypeTool, rec.ID, version.Version, model.VersionStatusOnline, version.Author, version.CommitMsg, string(definition), string(storage), nil, 0, millisTime(version.CreateTime, now), millisTime(rec.UpdateTime, now))
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) DeleteTool(ns, id string) error {
	if _, err := s.db.ExecContext(context.Background(), `DELETE FROM ai_resource_version WHERE namespace_id=? AND type=? AND name=?`, ns, resourceTypeTool, id); err != nil {
		return err
	}
	_, err := s.db.ExecContext(context.Background(), `DELETE FROM ai_resource WHERE namespace_id=? AND type=? AND name=?`, ns, resourceTypeTool, id)
	return err
}

func (s *Store) ListTools(ns string) ([]*model.ToolRecord, error) {
	rows, err := s.db.QueryContext(context.Background(), `SELECT name FROM ai_resource WHERE namespace_id=? AND type=? ORDER BY updated_at DESC`, ns, resourceTypeTool)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*model.ToolRecord{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		rec, err := s.LoadTool(ns, id)
		if err != nil {
			return nil, err
		}
		if rec != nil {
			out = append(out, rec)
		}
	}
	return out, rows.Err()
}

func (s *Store) AppendToolFailure(f *model.ToolFailureRecord) error {
	if f == nil {
		return nil
	}
	if f.ID == "" {
		f.ID = time.Now().Format("20060102150405.000000000")
	}
	if f.CreateTime == 0 {
		f.CreateTime = model.NowMillis()
	}
	metadata, _ := json.Marshal(f.Metadata)
	_, err := s.db.ExecContext(context.Background(), `INSERT INTO aihub_tool_failure(failure_id,app,namespace_id,object,tool_id,tool_version,agent_id,agent_version,runtime_id,session_id,run_id,trace_id,snapshot_id,attempt,error_code,error_message,retryable,input_digest,input_preview,duration_millis,metadata,reporter,created_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, f.ID, f.App, f.NamespaceID, nullable(f.Object), f.ToolID, nullable(f.ToolVersion), nullable(f.AgentID), nullable(f.AgentVersion), nullable(f.RuntimeID), nullable(f.SessionID), nullable(f.RunID), nullable(f.TraceID), nullable(f.SnapshotID), f.Attempt, nullable(f.ErrorCode), f.ErrorMessage, f.Retryable, nullable(f.InputDigest), nullable(f.InputPreview), f.DurationMillis, string(metadata), nullable(f.Reporter), millisTime(f.CreateTime, time.Now()))
	return err
}

func (s *Store) ListToolFailures(q model.ToolFailureQuery) ([]*model.ToolFailureRecord, int64, error) {
	query := `SELECT failure_id,app,namespace_id,object,tool_id,tool_version,agent_id,agent_version,runtime_id,session_id,run_id,trace_id,snapshot_id,attempt,error_code,error_message,retryable,input_digest,input_preview,duration_millis,metadata,reporter,UNIX_TIMESTAMP(created_at)*1000 FROM aihub_tool_failure WHERE 1=1`
	args := []interface{}{}
	if q.ToolID != "" {
		query += ` AND tool_id=?`
		args = append(args, q.ToolID)
	}
	if q.AgentID != "" {
		query += ` AND agent_id=?`
		args = append(args, q.AgentID)
	}
	if q.RuntimeID != "" {
		query += ` AND runtime_id=?`
		args = append(args, q.RuntimeID)
	}
	if q.SessionID != "" {
		query += ` AND session_id=?`
		args = append(args, q.SessionID)
	}
	if q.RunID != "" {
		query += ` AND run_id=?`
		args = append(args, q.RunID)
	}
	if q.TraceID != "" {
		query += ` AND trace_id=?`
		args = append(args, q.TraceID)
	}
	if q.SnapshotID != "" {
		query += ` AND snapshot_id=?`
		args = append(args, q.SnapshotID)
	}
	query += ` ORDER BY created_at DESC`
	if q.Limit <= 0 || q.Limit > 500 {
		q.Limit = 100
	}
	query += ` LIMIT ?`
	args = append(args, q.Limit)
	rows, err := s.db.QueryContext(context.Background(), query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []*model.ToolFailureRecord{}
	for rows.Next() {
		var f model.ToolFailureRecord
		var object, toolVersion, agentID, agentVersion, runtimeID, sessionID, runID, traceID, snapshotID, errorCode, inputDigest, inputPreview, metadata, reporter sql.NullString
		if err := rows.Scan(&f.ID, &f.App, &f.NamespaceID, &object, &f.ToolID, &toolVersion, &agentID, &agentVersion, &runtimeID, &sessionID, &runID, &traceID, &snapshotID, &f.Attempt, &errorCode, &f.ErrorMessage, &f.Retryable, &inputDigest, &inputPreview, &f.DurationMillis, &metadata, &reporter, &f.CreateTime); err != nil {
			return nil, 0, err
		}
		if object.Valid {
			f.Object = object.String
		}
		if toolVersion.Valid {
			f.ToolVersion = toolVersion.String
		}
		if agentID.Valid {
			f.AgentID = agentID.String
		}
		if agentVersion.Valid {
			f.AgentVersion = agentVersion.String
		}
		if runtimeID.Valid {
			f.RuntimeID = runtimeID.String
		}
		if sessionID.Valid {
			f.SessionID = sessionID.String
		}
		if runID.Valid {
			f.RunID = runID.String
		}
		if traceID.Valid {
			f.TraceID = traceID.String
		}
		if snapshotID.Valid {
			f.SnapshotID = snapshotID.String
		}
		if errorCode.Valid {
			f.ErrorCode = errorCode.String
		}
		if inputDigest.Valid {
			f.InputDigest = inputDigest.String
		}
		if inputPreview.Valid {
			f.InputPreview = inputPreview.String
		}
		if metadata.Valid && metadata.String != "" {
			_ = json.Unmarshal([]byte(metadata.String), &f.Metadata)
		}
		if reporter.Valid {
			f.Reporter = reporter.String
		}
		out = append(out, &f)
	}
	return out, int64(len(out)), rows.Err()
}

func (s *Store) LoadGroup(ns, name string) (*model.SkillGroup, error) {
	row := s.db.QueryRowContext(context.Background(), `SELECT display_name,description,scope,owner,labels,metadata,download_count,UNIX_TIMESTAMP(created_at)*1000,UNIX_TIMESTAMP(updated_at)*1000 FROM ai_resource_group WHERE namespace_id=? AND name=?`, ns, name)
	var g model.SkillGroup
	var labels, meta sql.NullString
	if err := row.Scan(&g.DisplayName, &g.Description, &g.Scope, &g.Owner, &labels, &meta, &g.DownloadCount, &g.CreateTime, &g.UpdateTime); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	g.NamespaceID = ns
	g.Name = name
	g.Labels = map[string]string{}
	g.Metadata = map[string]interface{}{}
	if labels.Valid && labels.String != "" {
		_ = json.Unmarshal([]byte(labels.String), &g.Labels)
	}
	if meta.Valid && meta.String != "" {
		_ = json.Unmarshal([]byte(meta.String), &g.Metadata)
	}
	rows, err := s.db.QueryContext(context.Background(), `SELECT resource_name,version,label,required_flag,sort_order FROM ai_resource_group_member WHERE namespace_id=? AND group_name=? ORDER BY sort_order ASC,id ASC`, ns, name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var m model.SkillGroupMember
		var ver, label sql.NullString
		if err := rows.Scan(&m.SkillName, &ver, &label, &m.Required, &m.Order); err != nil {
			return nil, err
		}
		if ver.Valid {
			m.Version = ver.String
		}
		if label.Valid {
			m.Label = label.String
		}
		g.Members = append(g.Members, m)
	}
	return &g, rows.Err()
}
func (s *Store) SaveGroup(g *model.SkillGroup) error {
	if g == nil {
		return nil
	}
	if g.Labels == nil {
		g.Labels = map[string]string{}
	}
	if g.Metadata == nil {
		g.Metadata = map[string]interface{}{}
	}
	labels, _ := json.Marshal(g.Labels)
	meta, _ := json.Marshal(g.Metadata)
	now := time.Now()
	_, err := s.db.ExecContext(context.Background(), `INSERT INTO ai_resource_group(namespace_id,name,display_name,description,status,scope,owner,labels,metadata,download_count,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?) ON DUPLICATE KEY UPDATE display_name=VALUES(display_name),description=VALUES(description),scope=VALUES(scope),owner=VALUES(owner),labels=VALUES(labels),metadata=VALUES(metadata),download_count=VALUES(download_count),updated_at=VALUES(updated_at)`, g.NamespaceID, g.Name, g.DisplayName, g.Description, model.MetaStatusEnable, g.Scope, g.Owner, string(labels), string(meta), g.DownloadCount, millisTime(g.CreateTime, now), millisTime(g.UpdateTime, now))
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(context.Background(), `DELETE FROM ai_resource_group_member WHERE namespace_id=? AND group_name=?`, g.NamespaceID, g.Name)
	if err != nil {
		return err
	}
	for i, m := range g.Members {
		if m.Order == 0 {
			m.Order = i + 1
		}
		_, err = s.db.ExecContext(context.Background(), `INSERT INTO ai_resource_group_member(namespace_id,group_name,resource_type,resource_name,version,label,required_flag,sort_order,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?)`, g.NamespaceID, g.Name, resourceTypeSkill, m.SkillName, nullable(m.Version), nullable(m.Label), m.Required, m.Order, now, now)
		if err != nil {
			return err
		}
	}
	return nil
}
func (s *Store) DeleteGroup(ns, name string) error {
	_, err := s.db.ExecContext(context.Background(), `DELETE FROM ai_resource_group_member WHERE namespace_id=? AND group_name=?`, ns, name)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(context.Background(), `DELETE FROM ai_resource_group WHERE namespace_id=? AND name=?`, ns, name)
	return err
}
func (s *Store) ListGroups(ns string) ([]*model.SkillGroup, error) {
	rows, err := s.db.QueryContext(context.Background(), `SELECT name FROM ai_resource_group WHERE namespace_id=? ORDER BY updated_at DESC`, ns)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*model.SkillGroup{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		g, err := s.LoadGroup(ns, name)
		if err != nil {
			return nil, err
		}
		if g != nil {
			out = append(out, g)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UpdateTime > out[j].UpdateTime })
	return out, rows.Err()
}
func millisTime(ms int64, f time.Time) time.Time {
	if ms <= 0 {
		return f
	}
	return time.UnixMilli(ms)
}
func nullable(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
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
	payload, _ := json.Marshal(e.Payload)
	res, err := s.db.ExecContext(context.Background(), `INSERT INTO aihub_catalog_event(app,event_type,object,resource_type,resource_id,skillset_name,version,revision,payload,created_at) VALUES(?,?,?,?,?,?,?,?,?,?)`, e.App, e.EventType, e.Object, e.ResourceType, e.ResourceID, nullable(e.SkillSetName), nullable(e.Version), nullable(e.Revision), string(payload), millisTime(e.CreatedAt, time.Now()))
	if err != nil {
		return err
	}
	if id, err := res.LastInsertId(); err == nil {
		e.ID = id
	}
	return nil
}

func (s *Store) ListCatalogEvents(q model.CatalogEventQuery) ([]*model.CatalogEvent, int64, error) {
	query := `SELECT id,app,event_type,object,resource_type,resource_id,skillset_name,version,revision,payload,UNIX_TIMESTAMP(created_at)*1000 FROM aihub_catalog_event WHERE 1=1`
	args := []interface{}{}
	if q.App != "" {
		query += ` AND app=?`
		args = append(args, q.App)
	}
	if q.SkillSetName != "" {
		query += ` AND (skillset_name=? OR skillset_name IS NULL OR skillset_name='')`
		args = append(args, q.SkillSetName)
	}
	if q.ResourceType != "" {
		query += ` AND resource_type=?`
		args = append(args, q.ResourceType)
	}
	if q.ResourceID != "" {
		query += ` AND resource_id=?`
		args = append(args, q.ResourceID)
	}
	if q.SinceID > 0 {
		query += ` AND id>?`
		args = append(args, q.SinceID)
	}
	query += ` ORDER BY id ASC`
	if q.Limit <= 0 || q.Limit > 500 {
		q.Limit = 100
	}
	query += ` LIMIT ?`
	args = append(args, q.Limit)
	rows, err := s.db.QueryContext(context.Background(), query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []*model.CatalogEvent{}
	for rows.Next() {
		var e model.CatalogEvent
		var skillset, version, revision, payload sql.NullString
		if err := rows.Scan(&e.ID, &e.App, &e.EventType, &e.Object, &e.ResourceType, &e.ResourceID, &skillset, &version, &revision, &payload, &e.CreatedAt); err != nil {
			return nil, 0, err
		}
		if skillset.Valid {
			e.SkillSetName = skillset.String
		}
		if version.Valid {
			e.Version = version.String
		}
		if revision.Valid {
			e.Revision = revision.String
		}
		if payload.Valid && payload.String != "" {
			_ = json.Unmarshal([]byte(payload.String), &e.Payload)
		}
		out = append(out, &e)
	}
	return out, int64(len(out)), rows.Err()
}

func (s *Store) LoadSandboxProfile(ns, id string) (*model.SandboxProfile, error) {
	row := s.db.QueryRowContext(context.Background(), `SELECT payload_json FROM aihub_sandbox_profile WHERE namespace_id=? AND id=?`, ns, id)
	var payload string
	if err := row.Scan(&payload); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	var p model.SandboxProfile
	if err := json.Unmarshal([]byte(payload), &p); err != nil {
		return nil, err
	}
	p.NamespaceID = ns
	if p.ID == "" {
		p.ID = id
	}
	return &p, nil
}
func (s *Store) SaveSandboxProfile(p *model.SandboxProfile) error {
	if p == nil {
		return nil
	}
	b, _ := json.Marshal(p)
	now := time.Now()
	_, err := s.db.ExecContext(context.Background(), `INSERT INTO aihub_sandbox_profile(namespace_id,id,payload_json,created_at,updated_at) VALUES(?,?,?,?,?) ON DUPLICATE KEY UPDATE payload_json=VALUES(payload_json),updated_at=VALUES(updated_at)`, p.NamespaceID, p.ID, string(b), millisTime(p.CreateTime, now), millisTime(p.UpdateTime, now))
	return err
}
func (s *Store) DeleteSandboxProfile(ns, id string) error {
	_, err := s.db.ExecContext(context.Background(), `DELETE FROM aihub_sandbox_profile WHERE namespace_id=? AND id=?`, ns, id)
	return err
}
func (s *Store) ListSandboxProfiles(ns string) ([]*model.SandboxProfile, error) {
	rows, err := s.db.QueryContext(context.Background(), `SELECT id FROM aihub_sandbox_profile WHERE namespace_id=? ORDER BY updated_at DESC`, ns)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*model.SandboxProfile{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		p, err := s.LoadSandboxProfile(ns, id)
		if err != nil {
			return nil, err
		}
		if p != nil {
			out = append(out, p)
		}
	}
	return out, rows.Err()
}
func (s *Store) LoadModelProfile(ns, id string) (*model.ModelProfile, error) {
	row := s.db.QueryRowContext(context.Background(), `SELECT payload_json FROM aihub_model_profile WHERE namespace_id=? AND id=?`, ns, id)
	var payload string
	if err := row.Scan(&payload); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	var p model.ModelProfile
	if err := json.Unmarshal([]byte(payload), &p); err != nil {
		return nil, err
	}
	p.NamespaceID = ns
	if p.ID == "" {
		p.ID = id
	}
	return &p, nil
}
func (s *Store) SaveModelProfile(p *model.ModelProfile) error {
	if p == nil {
		return nil
	}
	b, _ := json.Marshal(p)
	now := time.Now()
	_, err := s.db.ExecContext(context.Background(), `INSERT INTO aihub_model_profile(namespace_id,id,payload_json,created_at,updated_at) VALUES(?,?,?,?,?) ON DUPLICATE KEY UPDATE payload_json=VALUES(payload_json),updated_at=VALUES(updated_at)`, p.NamespaceID, p.ID, string(b), millisTime(p.CreateTime, now), millisTime(p.UpdateTime, now))
	return err
}
func (s *Store) DeleteModelProfile(ns, id string) error {
	_, err := s.db.ExecContext(context.Background(), `DELETE FROM aihub_model_profile WHERE namespace_id=? AND id=?`, ns, id)
	return err
}
func (s *Store) ListModelProfiles(ns string) ([]*model.ModelProfile, error) {
	rows, err := s.db.QueryContext(context.Background(), `SELECT id FROM aihub_model_profile WHERE namespace_id=? ORDER BY updated_at DESC`, ns)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*model.ModelProfile{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		p, err := s.LoadModelProfile(ns, id)
		if err != nil {
			return nil, err
		}
		if p != nil {
			out = append(out, p)
		}
	}
	return out, rows.Err()
}
func (s *Store) LoadSandboxPolicy(ns, id string) (*model.SandboxPolicy, error) {
	row := s.db.QueryRowContext(context.Background(), `SELECT payload_json FROM aihub_sandbox_policy WHERE namespace_id=? AND id=?`, ns, id)
	var payload string
	if err := row.Scan(&payload); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	var p model.SandboxPolicy
	if err := json.Unmarshal([]byte(payload), &p); err != nil {
		return nil, err
	}
	p.NamespaceID = ns
	if p.ID == "" {
		p.ID = id
	}
	return &p, nil
}
func (s *Store) SaveSandboxPolicy(p *model.SandboxPolicy) error {
	if p == nil {
		return nil
	}
	b, _ := json.Marshal(p)
	now := time.Now()
	_, err := s.db.ExecContext(context.Background(), `INSERT INTO aihub_sandbox_policy(namespace_id,id,payload_json,created_at,updated_at) VALUES(?,?,?,?,?) ON DUPLICATE KEY UPDATE payload_json=VALUES(payload_json),updated_at=VALUES(updated_at)`, p.NamespaceID, p.ID, string(b), millisTime(p.CreateTime, now), millisTime(p.UpdateTime, now))
	return err
}
func (s *Store) DeleteSandboxPolicy(ns, id string) error {
	_, err := s.db.ExecContext(context.Background(), `DELETE FROM aihub_sandbox_policy WHERE namespace_id=? AND id=?`, ns, id)
	return err
}
func (s *Store) ListSandboxPolicies(ns string) ([]*model.SandboxPolicy, error) {
	rows, err := s.db.QueryContext(context.Background(), `SELECT id FROM aihub_sandbox_policy WHERE namespace_id=? ORDER BY updated_at DESC`, ns)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*model.SandboxPolicy{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		p, err := s.LoadSandboxPolicy(ns, id)
		if err != nil {
			return nil, err
		}
		if p != nil {
			out = append(out, p)
		}
	}
	return out, rows.Err()
}

var _ store.Backend = (*Store)(nil)

func (s *Store) SaveProposal(p *model.SkillProposal) error {
	if p == nil {
		return nil
	}
	delta, _ := json.Marshal(p.Delta)
	evidence, _ := json.Marshal(p.Evidence)
	now := time.Now()
	_, err := s.db.ExecContext(context.Background(), `INSERT INTO ai_skill_proposal(proposal_id,namespace_id,skill_name,base_version,candidate_version,proposal_type,status,source_agent_id,source_session_id,source_run_id,source_task_id,reason,delta_json,evidence_json,overlay_ref,created_by,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?) ON DUPLICATE KEY UPDATE candidate_version=VALUES(candidate_version),proposal_type=VALUES(proposal_type),status=VALUES(status),source_agent_id=VALUES(source_agent_id),source_session_id=VALUES(source_session_id),source_run_id=VALUES(source_run_id),source_task_id=VALUES(source_task_id),reason=VALUES(reason),delta_json=VALUES(delta_json),evidence_json=VALUES(evidence_json),overlay_ref=VALUES(overlay_ref),created_by=VALUES(created_by),updated_at=VALUES(updated_at)`, p.ProposalID, p.NamespaceID, p.SkillName, p.BaseVersion, p.CandidateVersion, p.ProposalType, p.Status, nullable(p.Source.AgentID), nullable(p.Source.SessionID), nullable(p.Source.RunID), nullable(p.Source.TaskID), p.Reason, string(delta), string(evidence), p.OverlayRef, p.CreatedBy, millisTime(p.CreateTime, now), millisTime(p.UpdateTime, now))
	return err
}

func (s *Store) LoadProposal(proposalID string) (*model.SkillProposal, error) {
	row := s.db.QueryRowContext(context.Background(), `SELECT proposal_id,namespace_id,skill_name,base_version,candidate_version,proposal_type,status,source_agent_id,source_session_id,source_run_id,source_task_id,reason,delta_json,evidence_json,overlay_ref,created_by,UNIX_TIMESTAMP(created_at)*1000,UNIX_TIMESTAMP(updated_at)*1000 FROM ai_skill_proposal WHERE proposal_id=?`, proposalID)
	var p model.SkillProposal
	var cand, agent, sess, run, task, reason, delta, evidence, overlay, createdBy sql.NullString
	if err := row.Scan(&p.ProposalID, &p.NamespaceID, &p.SkillName, &p.BaseVersion, &cand, &p.ProposalType, &p.Status, &agent, &sess, &run, &task, &reason, &delta, &evidence, &overlay, &createdBy, &p.CreateTime, &p.UpdateTime); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if cand.Valid {
		p.CandidateVersion = cand.String
	}
	if agent.Valid {
		p.Source.AgentID = agent.String
	}
	if sess.Valid {
		p.Source.SessionID = sess.String
	}
	if run.Valid {
		p.Source.RunID = run.String
	}
	if task.Valid {
		p.Source.TaskID = task.String
	}
	if reason.Valid {
		p.Reason = reason.String
	}
	if overlay.Valid {
		p.OverlayRef = overlay.String
	}
	if createdBy.Valid {
		p.CreatedBy = createdBy.String
	}
	if delta.Valid && delta.String != "" {
		_ = json.Unmarshal([]byte(delta.String), &p.Delta)
	}
	if evidence.Valid && evidence.String != "" {
		_ = json.Unmarshal([]byte(evidence.String), &p.Evidence)
	}
	return &p, nil
}

func (s *Store) ListProposals(q model.ProposalQuery) ([]*model.SkillProposal, int64, error) {
	query := `SELECT proposal_id FROM ai_skill_proposal WHERE 1=1`
	args := []interface{}{}
	if q.NamespaceID != "" {
		query += ` AND namespace_id=?`
		args = append(args, q.NamespaceID)
	}
	if q.SkillName != "" {
		query += ` AND skill_name=?`
		args = append(args, q.SkillName)
	}
	if q.Status != "" {
		query += ` AND status=?`
		args = append(args, q.Status)
	}
	query += ` ORDER BY updated_at DESC`
	rows, err := s.db.QueryContext(context.Background(), query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []*model.SkillProposal{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, 0, err
		}
		p, err := s.LoadProposal(id)
		if err != nil {
			return nil, 0, err
		}
		if p != nil {
			out = append(out, p)
		}
	}
	return out, int64(len(out)), rows.Err()
}

func (s *Store) SaveOverlay(o *model.SkillOverlay) error {
	if o == nil {
		return nil
	}
	body, _ := json.Marshal(o.Overlay)
	var expires sql.NullTime
	if o.ExpiresAt > 0 {
		expires = sql.NullTime{Time: time.UnixMilli(o.ExpiresAt), Valid: true}
	}
	now := time.Now()
	_, err := s.db.ExecContext(context.Background(), `INSERT INTO ai_skill_overlay(overlay_ref,namespace_id,skill_name,base_version,proposal_id,overlay_json,status,expires_at,created_at) VALUES(?,?,?,?,?,?,?,?,?) ON DUPLICATE KEY UPDATE overlay_json=VALUES(overlay_json),status=VALUES(status),expires_at=VALUES(expires_at)`, o.OverlayRef, o.NamespaceID, o.SkillName, o.BaseVersion, o.ProposalID, string(body), o.Status, expires, millisTime(o.CreateTime, now))
	return err
}

func (s *Store) LoadOverlay(overlayRef string) (*model.SkillOverlay, error) {
	row := s.db.QueryRowContext(context.Background(), `SELECT overlay_ref,namespace_id,skill_name,base_version,proposal_id,overlay_json,status,IFNULL(UNIX_TIMESTAMP(expires_at)*1000,0),UNIX_TIMESTAMP(created_at)*1000 FROM ai_skill_overlay WHERE overlay_ref=?`, overlayRef)
	var o model.SkillOverlay
	var body string
	if err := row.Scan(&o.OverlayRef, &o.NamespaceID, &o.SkillName, &o.BaseVersion, &o.ProposalID, &body, &o.Status, &o.ExpiresAt, &o.CreateTime); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	_ = json.Unmarshal([]byte(body), &o.Overlay)
	return &o, nil
}

func (s *Store) SaveProposalValidation(v *model.ProposalValidation) error {
	if v == nil {
		return nil
	}
	check, _ := json.Marshal(v.CheckResult)
	test, _ := json.Marshal(v.TestResult)
	_, err := s.db.ExecContext(context.Background(), `INSERT INTO ai_skill_proposal_validation(proposal_id,validation_status,score,check_result,test_result,created_at) VALUES(?,?,?,?,?,?)`, v.ProposalID, v.ValidationStatus, v.Score, string(check), string(test), millisTime(v.CreateTime, time.Now()))
	return err
}

func (s *Store) ListProposalValidations(proposalID string) ([]*model.ProposalValidation, error) {
	rows, err := s.db.QueryContext(context.Background(), `SELECT proposal_id,validation_status,score,check_result,test_result,UNIX_TIMESTAMP(created_at)*1000 FROM ai_skill_proposal_validation WHERE proposal_id=? ORDER BY created_at DESC,id DESC`, proposalID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*model.ProposalValidation{}
	for rows.Next() {
		var v model.ProposalValidation
		var check, test sql.NullString
		if err := rows.Scan(&v.ProposalID, &v.ValidationStatus, &v.Score, &check, &test, &v.CreateTime); err != nil {
			return nil, err
		}
		if check.Valid && check.String != "" {
			_ = json.Unmarshal([]byte(check.String), &v.CheckResult)
		}
		if test.Valid && test.String != "" {
			_ = json.Unmarshal([]byte(test.String), &v.TestResult)
		}
		out = append(out, &v)
	}
	return out, rows.Err()
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
	meta, _ := json.Marshal(ns.Metadata)
	_, err := s.db.ExecContext(context.Background(), `INSERT INTO aihub_namespace(namespace_id,display_name,description,owner,visibility,metadata,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?) ON DUPLICATE KEY UPDATE display_name=VALUES(display_name),description=VALUES(description),owner=VALUES(owner),visibility=VALUES(visibility),metadata=VALUES(metadata),updated_at=VALUES(updated_at)`, ns.NamespaceID, ns.DisplayName, ns.Description, ns.Owner, ns.Visibility, string(meta), millisTime(ns.CreateTime, time.Now()), millisTime(ns.UpdateTime, time.Now()))
	return err
}
func (s *Store) LoadNamespace(namespaceID string) (*model.NamespaceInfo, error) {
	row := s.db.QueryRowContext(context.Background(), `SELECT namespace_id,display_name,description,owner,visibility,metadata,UNIX_TIMESTAMP(created_at)*1000,UNIX_TIMESTAMP(updated_at)*1000 FROM aihub_namespace WHERE namespace_id=?`, namespaceID)
	var ns model.NamespaceInfo
	var meta sql.NullString
	if err := row.Scan(&ns.NamespaceID, &ns.DisplayName, &ns.Description, &ns.Owner, &ns.Visibility, &meta, &ns.CreateTime, &ns.UpdateTime); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if meta.Valid && meta.String != "" {
		_ = json.Unmarshal([]byte(meta.String), &ns.Metadata)
	}
	if ns.Metadata == nil {
		ns.Metadata = map[string]interface{}{}
	}
	return &ns, nil
}
func (s *Store) ListNamespaces() ([]*model.NamespaceInfo, error) {
	rows, err := s.db.QueryContext(context.Background(), `SELECT namespace_id FROM aihub_namespace ORDER BY namespace_id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*model.NamespaceInfo{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ns, err := s.LoadNamespace(id)
		if err != nil {
			return nil, err
		}
		if ns != nil {
			out = append(out, ns)
		}
	}
	if len(out) == 0 {
		out = append(out, &model.NamespaceInfo{NamespaceID: model.DefaultNamespace, DisplayName: "Public", Visibility: model.ScopePublic, CreateTime: model.NowMillis(), UpdateTime: model.NowMillis(), Metadata: map[string]interface{}{}})
	}
	return out, rows.Err()
}
func (s *Store) SaveNamespaceMember(m *model.NamespaceMember) error {
	if m == nil {
		return nil
	}
	if m.CreateTime == 0 {
		m.CreateTime = model.NowMillis()
	}
	m.UpdateTime = model.NowMillis()
	roles, _ := json.Marshal(m.Roles)
	_, err := s.db.ExecContext(context.Background(), `INSERT INTO aihub_namespace_member(namespace_id,subject_id,subject_type,display_name,roles,created_at,updated_at) VALUES(?,?,?,?,?,?,?) ON DUPLICATE KEY UPDATE subject_type=VALUES(subject_type),display_name=VALUES(display_name),roles=VALUES(roles),updated_at=VALUES(updated_at)`, m.NamespaceID, m.SubjectID, m.SubjectType, m.DisplayName, string(roles), millisTime(m.CreateTime, time.Now()), millisTime(m.UpdateTime, time.Now()))
	return err
}
func (s *Store) DeleteNamespaceMember(namespaceID, subjectID string) error {
	_, err := s.db.ExecContext(context.Background(), `DELETE FROM aihub_namespace_member WHERE namespace_id=? AND subject_id=?`, namespaceID, subjectID)
	return err
}
func (s *Store) ListNamespaceMembers(q model.NamespaceMemberQuery) ([]*model.NamespaceMember, int64, error) {
	query := `SELECT namespace_id,subject_id,subject_type,display_name,roles,UNIX_TIMESTAMP(created_at)*1000,UNIX_TIMESTAMP(updated_at)*1000 FROM aihub_namespace_member WHERE namespace_id=?`
	args := []interface{}{q.NamespaceID}
	if q.SubjectID != "" {
		query += ` AND subject_id=?`
		args = append(args, q.SubjectID)
	}
	query += ` ORDER BY subject_id ASC`
	rows, err := s.db.QueryContext(context.Background(), query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []*model.NamespaceMember{}
	for rows.Next() {
		var m model.NamespaceMember
		var roles string
		if err := rows.Scan(&m.NamespaceID, &m.SubjectID, &m.SubjectType, &m.DisplayName, &roles, &m.CreateTime, &m.UpdateTime); err != nil {
			return nil, 0, err
		}
		_ = json.Unmarshal([]byte(roles), &m.Roles)
		out = append(out, &m)
	}
	return out, int64(len(out)), rows.Err()
}
func (s *Store) SetStar(namespaceID, skillName, subjectID string, starred bool) error {
	if starred {
		_, err := s.db.ExecContext(context.Background(), `INSERT IGNORE INTO aihub_star(namespace_id,skill_name,subject_id,created_at) VALUES(?,?,?,?)`, namespaceID, skillName, subjectID, time.Now())
		return err
	}
	_, err := s.db.ExecContext(context.Background(), `DELETE FROM aihub_star WHERE namespace_id=? AND skill_name=? AND subject_id=?`, namespaceID, skillName, subjectID)
	return err
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
	_, err := s.db.ExecContext(context.Background(), `INSERT INTO aihub_rating(namespace_id,skill_name,subject_id,rating,comment,created_at,updated_at) VALUES(?,?,?,?,?,?,?) ON DUPLICATE KEY UPDATE rating=VALUES(rating),comment=VALUES(comment),updated_at=VALUES(updated_at)`, r.NamespaceID, r.SkillName, r.SubjectID, r.Rating, r.Comment, millisTime(r.CreateTime, time.Now()), millisTime(r.UpdateTime, time.Now()))
	return err
}
func (s *Store) SetSubscription(namespaceID, targetType, targetName, subjectID string, subscribed bool) error {
	if subscribed {
		_, err := s.db.ExecContext(context.Background(), `INSERT IGNORE INTO aihub_subscription(namespace_id,target_type,target_name,subject_id,created_at) VALUES(?,?,?,?,?)`, namespaceID, targetType, targetName, subjectID, time.Now())
		return err
	}
	_, err := s.db.ExecContext(context.Background(), `DELETE FROM aihub_subscription WHERE namespace_id=? AND target_type=? AND target_name=? AND subject_id=?`, namespaceID, targetType, targetName, subjectID)
	return err
}
func (s *Store) ListSubscribers(namespaceID, targetType, targetName string) ([]string, error) {
	rows, err := s.db.QueryContext(context.Background(), `SELECT subject_id FROM aihub_subscription WHERE namespace_id=? AND target_type=? AND target_name=?`, namespaceID, targetType, targetName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func (s *Store) GetSkillSocialStats(namespaceID, skillName, subjectID string) (*model.SkillSocialStats, error) {
	st := &model.SkillSocialStats{NamespaceID: namespaceID, SkillName: skillName}
	_ = s.db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM aihub_star WHERE namespace_id=? AND skill_name=?`, namespaceID, skillName).Scan(&st.Stars)
	_ = s.db.QueryRowContext(context.Background(), `SELECT COUNT(*),IFNULL(AVG(rating),0) FROM aihub_rating WHERE namespace_id=? AND skill_name=?`, namespaceID, skillName).Scan(&st.RatingCount, &st.RatingAverage)
	_ = s.db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM aihub_subscription WHERE namespace_id=? AND target_type='skill' AND target_name=?`, namespaceID, skillName).Scan(&st.Subscribers)
	if subjectID != "" {
		var n int
		_ = s.db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM aihub_star WHERE namespace_id=? AND skill_name=? AND subject_id=?`, namespaceID, skillName, subjectID).Scan(&n)
		st.MyStarred = n > 0
		n = 0
		_ = s.db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM aihub_subscription WHERE namespace_id=? AND target_type='skill' AND target_name=? AND subject_id=?`, namespaceID, skillName, subjectID).Scan(&n)
		st.MySubscribed = n > 0
		_ = s.db.QueryRowContext(context.Background(), `SELECT rating FROM aihub_rating WHERE namespace_id=? AND skill_name=? AND subject_id=?`, namespaceID, skillName, subjectID).Scan(&st.MyRating)
	}
	return st, nil
}
func (s *Store) AppendAudit(l *model.AuditLog) error {
	if l == nil {
		return nil
	}
	if l.ID == "" {
		l.ID = time.Now().Format("20060102150405.000000000")
	}
	if l.CreateTime == 0 {
		l.CreateTime = model.NowMillis()
	}
	detail, _ := json.Marshal(l.Detail)
	_, err := s.db.ExecContext(context.Background(), `INSERT INTO aihub_audit_log(log_id,namespace_id,resource_type,resource_name,version,action,operator,detail,request_id,created_at) VALUES(?,?,?,?,?,?,?,?,?,?)`, l.ID, l.NamespaceID, l.ResourceType, l.ResourceName, l.Version, l.Action, l.Operator, string(detail), l.RequestID, millisTime(l.CreateTime, time.Now()))
	return err
}
func (s *Store) ListAuditLogs(q model.AuditQuery) ([]*model.AuditLog, int64, error) {
	query := `SELECT log_id,namespace_id,resource_type,resource_name,version,action,operator,detail,request_id,UNIX_TIMESTAMP(created_at)*1000 FROM aihub_audit_log WHERE 1=1`
	args := []interface{}{}
	if q.NamespaceID != "" {
		query += ` AND namespace_id=?`
		args = append(args, q.NamespaceID)
	}
	if q.ResourceType != "" {
		query += ` AND resource_type=?`
		args = append(args, q.ResourceType)
	}
	if q.ResourceName != "" {
		query += ` AND resource_name=?`
		args = append(args, q.ResourceName)
	}
	if q.Action != "" {
		query += ` AND action=?`
		args = append(args, q.Action)
	}
	if q.Operator != "" {
		query += ` AND operator=?`
		args = append(args, q.Operator)
	}
	query += ` ORDER BY created_at DESC LIMIT 500`
	rows, err := s.db.QueryContext(context.Background(), query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []*model.AuditLog{}
	for rows.Next() {
		var l model.AuditLog
		var detail sql.NullString
		if err := rows.Scan(&l.ID, &l.NamespaceID, &l.ResourceType, &l.ResourceName, &l.Version, &l.Action, &l.Operator, &detail, &l.RequestID, &l.CreateTime); err != nil {
			return nil, 0, err
		}
		if detail.Valid && detail.String != "" {
			_ = json.Unmarshal([]byte(detail.String), &l.Detail)
		}
		out = append(out, &l)
	}
	return out, int64(len(out)), rows.Err()
}
func (s *Store) SaveToken(t *model.TokenInfo) error {
	if t == nil {
		return nil
	}
	roles, _ := json.Marshal(t.Roles)
	perms, _ := json.Marshal(t.Permissions)
	nss, _ := json.Marshal(t.Namespaces)
	if t.CreateTime == 0 {
		t.CreateTime = model.NowMillis()
	}
	if t.Status == "" {
		t.Status = "active"
	}
	var exp sql.NullTime
	if t.ExpiresAt > 0 {
		exp = sql.NullTime{Time: time.UnixMilli(t.ExpiresAt), Valid: true}
	}
	_, err := s.db.ExecContext(context.Background(), `INSERT INTO aihub_token(key_id,name,subject_id,subject_type,roles,permissions,namespaces,status,token_hash,expires_at,created_at) VALUES(?,?,?,?,?,?,?,?,?,?,?) ON DUPLICATE KEY UPDATE name=VALUES(name),roles=VALUES(roles),permissions=VALUES(permissions),namespaces=VALUES(namespaces),status=VALUES(status),token_hash=VALUES(token_hash),expires_at=VALUES(expires_at)`, t.KeyID, t.Name, t.SubjectID, t.SubjectType, string(roles), string(perms), string(nss), t.Status, t.TokenHash, exp, millisTime(t.CreateTime, time.Now()))
	return err
}
func (s *Store) DeleteToken(keyID string) error {
	_, err := s.db.ExecContext(context.Background(), `UPDATE aihub_token SET status='deleted' WHERE key_id=?`, keyID)
	return err
}
func (s *Store) ListTokens(subjectID string) ([]*model.TokenInfo, error) {
	query := `SELECT key_id,name,subject_id,subject_type,roles,permissions,namespaces,status,IFNULL(UNIX_TIMESTAMP(expires_at)*1000,0),IFNULL(UNIX_TIMESTAMP(last_used_at)*1000,0),UNIX_TIMESTAMP(created_at)*1000 FROM aihub_token WHERE status<>'deleted'`
	args := []interface{}{}
	if subjectID != "" {
		query += ` AND subject_id=?`
		args = append(args, subjectID)
	}
	rows, err := s.db.QueryContext(context.Background(), query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*model.TokenInfo{}
	for rows.Next() {
		var t model.TokenInfo
		var roles, perms, nss string
		if err := rows.Scan(&t.KeyID, &t.Name, &t.SubjectID, &t.SubjectType, &roles, &perms, &nss, &t.Status, &t.ExpiresAt, &t.LastUsedAt, &t.CreateTime); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(roles), &t.Roles)
		_ = json.Unmarshal([]byte(perms), &t.Permissions)
		_ = json.Unmarshal([]byte(nss), &t.Namespaces)
		out = append(out, &t)
	}
	return out, rows.Err()
}
func (s *Store) SaveIdempotency(r *model.IdempotencyRecord) error {
	if r == nil {
		return nil
	}
	_, err := s.db.ExecContext(context.Background(), `INSERT INTO aihub_idempotency(idempotency_key,method,path,request_hash,status_code,response_body,created_at,expires_at) VALUES(?,?,?,?,?,?,?,?) ON DUPLICATE KEY UPDATE status_code=VALUES(status_code),response_body=VALUES(response_body)`, r.Key, r.Method, r.Path, r.RequestHash, r.StatusCode, r.ResponseBody, millisTime(r.CreateTime, time.Now()), millisTime(r.ExpiresAt, time.Now().Add(time.Hour)))
	return err
}
func (s *Store) LoadIdempotency(key string) (*model.IdempotencyRecord, error) {
	row := s.db.QueryRowContext(context.Background(), `SELECT idempotency_key,method,path,request_hash,status_code,response_body,UNIX_TIMESTAMP(created_at)*1000,UNIX_TIMESTAMP(expires_at)*1000 FROM aihub_idempotency WHERE idempotency_key=? AND expires_at>NOW()`, key)
	var r model.IdempotencyRecord
	if err := row.Scan(&r.Key, &r.Method, &r.Path, &r.RequestHash, &r.StatusCode, &r.ResponseBody, &r.CreateTime, &r.ExpiresAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &r, nil
}

func (s *Store) FindActiveTokenByHash(tokenHash string) (*model.TokenInfo, error) {
	if tokenHash == "" {
		return nil, nil
	}
	row := s.db.QueryRowContext(context.Background(), `SELECT key_id,name,subject_id,subject_type,roles,permissions,namespaces,status,IFNULL(UNIX_TIMESTAMP(expires_at)*1000,0),IFNULL(UNIX_TIMESTAMP(last_used_at)*1000,0),UNIX_TIMESTAMP(created_at)*1000 FROM aihub_token WHERE token_hash=? AND status='active'`, tokenHash)
	var t model.TokenInfo
	var roles, perms, nss string
	if err := row.Scan(&t.KeyID, &t.Name, &t.SubjectID, &t.SubjectType, &roles, &perms, &nss, &t.Status, &t.ExpiresAt, &t.LastUsedAt, &t.CreateTime); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if t.ExpiresAt > 0 && t.ExpiresAt < model.NowMillis() {
		return nil, nil
	}
	_ = json.Unmarshal([]byte(roles), &t.Roles)
	_ = json.Unmarshal([]byte(perms), &t.Permissions)
	_ = json.Unmarshal([]byte(nss), &t.Namespaces)
	_, _ = s.db.ExecContext(context.Background(), `UPDATE aihub_token SET last_used_at=? WHERE key_id=?`, time.Now(), t.KeyID)
	return &t, nil
}

func (s *Store) AppendNotification(n *model.Notification) error {
	if n == nil {
		return nil
	}
	payload, _ := json.Marshal(n.Payload)
	_, err := s.db.ExecContext(context.Background(), `INSERT INTO aihub_notification(notification_id,namespace_id,subject_id,target_type,target_name,event_type,title,message,payload,read_flag,created_at) VALUES(?,?,?,?,?,?,?,?,?,?,?)`, n.ID, n.NamespaceID, n.SubjectID, n.TargetType, n.TargetName, n.EventType, n.Title, n.Message, string(payload), n.Read, millisTime(n.CreateTime, time.Now()))
	return err
}
func (s *Store) ListNotifications(q model.NotificationQuery) ([]*model.Notification, int64, error) {
	query := `SELECT notification_id,namespace_id,subject_id,target_type,target_name,event_type,title,message,payload,read_flag,UNIX_TIMESTAMP(created_at)*1000 FROM aihub_notification WHERE subject_id=?`
	args := []interface{}{q.SubjectID}
	if q.UnreadOnly {
		query += ` AND read_flag=FALSE`
	}
	query += ` ORDER BY created_at DESC LIMIT ? OFFSET ?`
	limit := q.PageSize
	if limit <= 0 {
		limit = 50
	}
	offset := 0
	if q.PageNo > 1 {
		offset = (q.PageNo - 1) * limit
	}
	args = append(args, limit, offset)
	rows, err := s.db.QueryContext(context.Background(), query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []*model.Notification{}
	for rows.Next() {
		var n model.Notification
		var payload sql.NullString
		if err := rows.Scan(&n.ID, &n.NamespaceID, &n.SubjectID, &n.TargetType, &n.TargetName, &n.EventType, &n.Title, &n.Message, &payload, &n.Read, &n.CreateTime); err != nil {
			return nil, 0, err
		}
		if payload.Valid && payload.String != "" {
			_ = json.Unmarshal([]byte(payload.String), &n.Payload)
		}
		out = append(out, &n)
	}
	return out, int64(len(out)), rows.Err()
}
func (s *Store) MarkNotificationRead(subjectID, notificationID string) error {
	_, err := s.db.ExecContext(context.Background(), `UPDATE aihub_notification SET read_flag=TRUE WHERE subject_id=? AND notification_id=?`, subjectID, notificationID)
	return err
}
