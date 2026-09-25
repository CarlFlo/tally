package flows

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/CarlFlo/tally/internal/automationchain"
	"github.com/CarlFlo/tally/internal/database"
	"github.com/CarlFlo/tally/internal/settings"
)

type Store struct{ DB *database.Store }

func defaultName(id string) string {
	if id == automationchain.DefaultAnimatedID {
		return "Anime / cartoon default"
	}
	return "Live action default"
}

// EnsureDefaults captures the current automation policy when defaults are first created.
// Existing installations keep their effective choices until an administrator edits a chain.
func (s Store) EnsureDefaults(ctx context.Context) error {
	config := settings.DefaultTorrentAutomation()
	if _, err := (settings.Store{DB: s.DB}).Load(ctx, "torrent_automation", &config); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	config = config.Effective()
	for _, item := range []struct{ id, profile string }{{automationchain.DefaultLiveID, "live"}, {automationchain.DefaultAnimatedID, "animated"}} {
		d := DefaultProfileDefinition(item.profile, config)
		if err := Validate(defaultName(item.id), d); err != nil {
			return err
		}
		raw, err := json.Marshal(d)
		if err != nil {
			return err
		}
		tx, err := s.DB.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		res, err := tx.ExecContext(ctx, `INSERT INTO automation_flows(id,name,revision,created_at,updated_at) VALUES(?,?,1,unixepoch(),unixepoch()) ON CONFLICT(id) DO NOTHING`, item.id, defaultName(item.id))
		if err != nil {
			tx.Rollback()
			return err
		}
		inserted, err := res.RowsAffected()
		if err != nil {
			tx.Rollback()
			return err
		}
		if inserted == 1 {
			if _, err = tx.ExecContext(ctx, `INSERT INTO automation_flow_revisions(flow_id,revision,definition,created_at) VALUES(?,1,?,unixepoch())`, item.id, string(raw)); err != nil {
				tx.Rollback()
				return err
			}
		}
		if err = tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

func (s Store) ResetDefault(ctx context.Context, id string, expectedRevision int) (Flow, error) {
	if !automationchain.IsDefault(id) {
		return Flow{}, errors.New("only default chains can be reset")
	}
	if err := s.EnsureDefaults(ctx); err != nil {
		return Flow{}, err
	}
	flow, err := s.Get(ctx, id)
	if err != nil {
		return Flow{}, err
	}
	if flow.Revision != expectedRevision {
		return Flow{}, errors.New("flow changed since it was loaded")
	}
	profile := "live"
	if id == automationchain.DefaultAnimatedID {
		profile = "animated"
	}
	flow.Name = defaultName(id)
	flow.Definition = DefaultProfileDefinition(profile, settings.DefaultTorrentAutomation())
	return s.Save(ctx, flow, flow.Revision)
}

func nullable(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func (s Store) Save(ctx context.Context, flow Flow, expectedRevision int) (Flow, error) {
	if err := Validate(flow.Name, flow.Definition); err != nil {
		return Flow{}, err
	}
	if flow.ShowID != "" {
		var exists int
		if err := s.DB.QueryRowContext(ctx, "SELECT 1 FROM shows WHERE id=?", flow.ShowID).Scan(&exists); err != nil {
			return Flow{}, err
		}
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return Flow{}, err
	}
	defer tx.Rollback()
	now := time.Now().Unix()
	if flow.ID == "" {
		flow.ID = database.ID()
		flow.Revision = 1
		flow.CreatedAt = now
		_, err = tx.ExecContext(ctx, "INSERT INTO automation_flows(id,name,show_id,revision,created_at,updated_at) VALUES(?,?,?,?,?,?)", flow.ID, flow.Name, nullable(flow.ShowID), flow.Revision, now, now)
	} else {
		if !validID(flow.ID) {
			return Flow{}, errors.New("invalid flow ID")
		}
		var current int
		var savedShowID sql.NullString
		if err = tx.QueryRowContext(ctx, "SELECT revision,show_id FROM automation_flows WHERE id=?", flow.ID).Scan(&current, &savedShowID); err != nil {
			return Flow{}, err
		}
		if savedShowID.String != flow.ShowID {
			return Flow{}, errors.New("chain scope cannot be changed")
		}
		if automationchain.IsDefault(flow.ID) && flow.Name != defaultName(flow.ID) {
			return Flow{}, errors.New("default chain name cannot be changed")
		}
		if current != expectedRevision {
			return Flow{}, fmt.Errorf("flow changed since it was loaded")
		}
		flow.Revision = current + 1
		_, err = tx.ExecContext(ctx, "UPDATE automation_flows SET name=?,revision=?,updated_at=? WHERE id=? AND revision=? AND COALESCE(show_id,'')=?", flow.Name, flow.Revision, now, flow.ID, current, flow.ShowID)
	}
	if err != nil {
		return Flow{}, err
	}
	flow.UpdatedAt = now
	definition, err := json.Marshal(flow.Definition)
	if err != nil {
		return Flow{}, err
	}
	// Keep one saved definition per flow. Replay records already carry their own
	// immutable definition snapshot, so older editable revisions are redundant.
	if _, err = tx.ExecContext(ctx, "DELETE FROM automation_flow_revisions WHERE flow_id=?", flow.ID); err != nil {
		return Flow{}, err
	}
	_, err = tx.ExecContext(ctx, "INSERT INTO automation_flow_revisions(flow_id,revision,definition,created_at) VALUES(?,?,?,?)", flow.ID, flow.Revision, string(definition), now)
	if err != nil {
		return Flow{}, err
	}
	if err = tx.Commit(); err != nil {
		return Flow{}, err
	}
	return flow, nil
}
func (s Store) Get(ctx context.Context, id string) (Flow, error) {
	if automationchain.IsDefault(id) {
		if err := s.EnsureDefaults(ctx); err != nil {
			return Flow{}, err
		}
	}
	var flow Flow
	var raw string
	var showID sql.NullString
	err := s.DB.QueryRowContext(ctx, `SELECT f.id,f.name,f.show_id,f.revision,r.definition,f.created_at,f.updated_at
		FROM automation_flows f JOIN automation_flow_revisions r ON r.flow_id=f.id AND r.revision=f.revision WHERE f.id=?`, id).
		Scan(&flow.ID, &flow.Name, &showID, &flow.Revision, &raw, &flow.CreatedAt, &flow.UpdatedAt)
	if err != nil {
		return Flow{}, err
	}
	flow.ShowID = showID.String
	flow.Definition, err = decodeDefinition(raw)
	return flow, err
}

func (s Store) Delete(ctx context.Context, id string) error {
	if !validID(id) {
		return errors.New("invalid flow ID")
	}
	if automationchain.IsDefault(id) {
		return errors.New("default chain cannot be deleted")
	}
	result, err := s.DB.ExecContext(ctx, "DELETE FROM automation_flows WHERE id=?", id)
	if err != nil {
		return err
	}
	deleted, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if deleted == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s Store) List(ctx context.Context) ([]Flow, error) {
	if err := s.EnsureDefaults(ctx); err != nil {
		return nil, err
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT f.id,f.name,f.show_id,f.revision,r.definition,f.created_at,f.updated_at
		FROM automation_flows f JOIN automation_flow_revisions r ON r.flow_id=f.id AND r.revision=f.revision ORDER BY f.updated_at DESC,f.id LIMIT 100`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Flow, 0)
	for rows.Next() {
		var flow Flow
		var raw string
		var showID sql.NullString
		if err = rows.Scan(&flow.ID, &flow.Name, &showID, &flow.Revision, &raw, &flow.CreatedAt, &flow.UpdatedAt); err != nil {
			return nil, err
		}
		flow.ShowID = showID.String
		if flow.Definition, err = decodeDefinition(raw); err != nil {
			return nil, err
		}
		out = append(out, flow)
	}
	return out, rows.Err()
}

// Assignment is deployment-global, like the downloader and show enrollment.
func (s Store) Assigned(ctx context.Context, showID string) (string, error) {
	var id string
	err := s.DB.QueryRowContext(ctx, "SELECT flow_id FROM torrent_show_chain WHERE show_id=?", showID).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return id, err
}
func (s Store) Assign(ctx context.Context, showID, flowID string) error {
	if flowID == "" {
		_, err := s.DB.ExecContext(ctx, "DELETE FROM torrent_show_chain WHERE show_id=?", showID)
		return err
	}
	if !validID(flowID) {
		return errors.New("invalid chain ID")
	}
	flow, err := s.Get(ctx, flowID)
	if err != nil {
		return err
	}
	if flow.ShowID != "" && flow.ShowID != showID {
		return errors.New("chain is not available for this show")
	}
	if err := Validate(flow.Name, flow.Definition); err != nil {
		return err
	}
	var raw string
	if err := s.DB.QueryRowContext(ctx, "SELECT definition FROM automation_flow_revisions WHERE flow_id=? AND revision=?", flowID, flow.Revision).Scan(&raw); err != nil {
		return err
	}
	var current Definition
	if err := json.Unmarshal([]byte(raw), &current); err != nil {
		return err
	}
	if len(current.Blocks) == 0 {
		flow, err = s.Save(ctx, flow, flow.Revision)
		if err != nil {
			return err
		}
	}
	result, err := s.DB.ExecContext(ctx, `INSERT INTO torrent_show_chain(show_id,flow_id)
		SELECT ?,id FROM automation_flows WHERE id=? AND (show_id IS NULL OR show_id=?)
		ON CONFLICT(show_id) DO UPDATE SET flow_id=excluded.flow_id`, showID, flowID, showID)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return errors.New("chain is not available for this show")
	}
	return nil
}
func (s Store) Record(ctx context.Context, run Run) (Run, error) {
	if run.FlowID == "" || run.FlowRevision < 0 {
		return Run{}, errors.New("invalid run flow")
	}
	run.ID = database.ID()
	event, err := json.Marshal(run.Event)
	if err != nil {
		return Run{}, err
	}
	definition, err := json.Marshal(run.Definition)
	if err != nil {
		return Run{}, err
	}
	steps, err := json.Marshal(run.Steps)
	if err != nil {
		return Run{}, err
	}
	if len(event) > 4096 || len(definition) > 65536 || len(steps) > 131072 {
		return Run{}, errors.New("run snapshot is too large")
	}
	var source any
	if run.SourceRunID != "" {
		source = run.SourceRunID
	}
	_, err = s.DB.ExecContext(ctx, `INSERT INTO automation_flow_runs(id,flow_id,flow_revision,source_run_id,trigger_event,definition,trace,status,started_at,duration_ms)
		VALUES(?,?,?,?,?,?,?,?,?,?)`, run.ID, run.FlowID, run.FlowRevision, source, string(event), string(definition), string(steps), run.Status, run.StartedAt, run.DurationMS)
	if err != nil {
		return Run{}, err
	}
	// Keep a bounded detailed history per flow. Revision snapshots remain durable.
	_, err = s.DB.ExecContext(ctx, `DELETE FROM automation_flow_runs WHERE flow_id=? AND id NOT IN
		(SELECT id FROM automation_flow_runs WHERE flow_id=? ORDER BY started_at DESC,id DESC LIMIT 200)`, run.FlowID, run.FlowID)
	return run, err
}
func (s Store) GetRun(ctx context.Context, id string) (Run, error) {
	var run Run
	var event, definition, trace string
	var source sql.NullString
	err := s.DB.QueryRowContext(ctx, `SELECT id,flow_id,flow_revision,source_run_id,trigger_event,definition,trace,status,started_at,duration_ms FROM automation_flow_runs WHERE id=?`, id).
		Scan(&run.ID, &run.FlowID, &run.FlowRevision, &source, &event, &definition, &trace, &run.Status, &run.StartedAt, &run.DurationMS)
	if err != nil {
		return Run{}, err
	}
	run.SourceRunID = source.String
	if err = json.Unmarshal([]byte(event), &run.Event); err != nil {
		return Run{}, err
	}
	if run.Definition, err = decodeDefinition(definition); err != nil {
		return Run{}, err
	}
	if err = json.Unmarshal([]byte(trace), &run.Steps); err != nil {
		return Run{}, err
	}
	var old struct {
		Nodes []struct {
			ID   string `json:"id"`
			Type string `json:"type"`
		} `json:"nodes"`
	}
	if err = json.Unmarshal([]byte(definition), &old); err != nil {
		return Run{}, err
	}
	if len(old.Nodes) > 0 {
		ids := map[string]string{}
		for _, node := range old.Nodes {
			for _, block := range run.Definition.Blocks {
				if node.Type == block.Type {
					ids[node.ID] = block.ID
					break
				}
			}
		}
		for i := range run.Steps {
			if id := ids[run.Steps[i].NodeID]; id != "" {
				run.Steps[i].NodeID = id
			}
		}
	}
	return run, nil
}

type RunSummary struct {
	ID           string `json:"id"`
	FlowID       string `json:"flow_id"`
	FlowRevision int    `json:"flow_revision"`
	SourceRunID  string `json:"source_run_id,omitempty"`
	Event        Event  `json:"event"`
	Status       string `json:"status"`
	StartedAt    int64  `json:"started_at"`
	DurationMS   int64  `json:"duration_ms"`
}

func (s Store) ListRuns(ctx context.Context) ([]RunSummary, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT id,flow_id,flow_revision,source_run_id,trigger_event,status,started_at,duration_ms FROM automation_flow_runs ORDER BY started_at DESC,id DESC LIMIT 100`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]RunSummary, 0)
	for rows.Next() {
		var item RunSummary
		var source sql.NullString
		var event string
		if err = rows.Scan(&item.ID, &item.FlowID, &item.FlowRevision, &source, &event, &item.Status, &item.StartedAt, &item.DurationMS); err != nil {
			return nil, err
		}
		item.SourceRunID = source.String
		if err = json.Unmarshal([]byte(event), &item.Event); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}
