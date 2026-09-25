package flows

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/CarlFlo/tally/internal/database"
)

type Store struct{ DB *database.Store }

func (s Store) Save(ctx context.Context, flow Flow, expectedRevision int) (Flow, error) {
	if err := Validate(flow.Name, flow.Definition); err != nil {
		return Flow{}, err
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
		_, err = tx.ExecContext(ctx, "INSERT INTO automation_flows(id,name,revision,created_at,updated_at) VALUES(?,?,?,?,?)", flow.ID, flow.Name, flow.Revision, now, now)
	} else {
		if !validID(flow.ID) {
			return Flow{}, errors.New("invalid flow ID")
		}
		var current int
		if err = tx.QueryRowContext(ctx, "SELECT revision FROM automation_flows WHERE id=?", flow.ID).Scan(&current); err != nil {
			return Flow{}, err
		}
		if current != expectedRevision {
			return Flow{}, fmt.Errorf("flow changed since it was loaded")
		}
		flow.Revision = current + 1
		_, err = tx.ExecContext(ctx, "UPDATE automation_flows SET name=?,revision=?,updated_at=? WHERE id=? AND revision=?", flow.Name, flow.Revision, now, flow.ID, current)
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
	var flow Flow
	var raw string
	err := s.DB.QueryRowContext(ctx, `SELECT f.id,f.name,f.revision,r.definition,f.created_at,f.updated_at
		FROM automation_flows f JOIN automation_flow_revisions r ON r.flow_id=f.id AND r.revision=f.revision WHERE f.id=?`, id).
		Scan(&flow.ID, &flow.Name, &flow.Revision, &raw, &flow.CreatedAt, &flow.UpdatedAt)
	if err != nil {
		return Flow{}, err
	}
	err = json.Unmarshal([]byte(raw), &flow.Definition)
	return flow, err
}

func (s Store) Delete(ctx context.Context, id string) error {
	if !validID(id) {
		return errors.New("invalid flow ID")
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
	rows, err := s.DB.QueryContext(ctx, `SELECT f.id,f.name,f.revision,r.definition,f.created_at,f.updated_at
		FROM automation_flows f JOIN automation_flow_revisions r ON r.flow_id=f.id AND r.revision=f.revision ORDER BY f.updated_at DESC,f.id LIMIT 100`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Flow, 0)
	for rows.Next() {
		var flow Flow
		var raw string
		if err = rows.Scan(&flow.ID, &flow.Name, &flow.Revision, &raw, &flow.CreatedAt, &flow.UpdatedAt); err != nil {
			return nil, err
		}
		if err = json.Unmarshal([]byte(raw), &flow.Definition); err != nil {
			return nil, err
		}
		out = append(out, flow)
	}
	return out, rows.Err()
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
	if err = json.Unmarshal([]byte(definition), &run.Definition); err != nil {
		return Run{}, err
	}
	if err = json.Unmarshal([]byte(trace), &run.Steps); err != nil {
		return Run{}, err
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
