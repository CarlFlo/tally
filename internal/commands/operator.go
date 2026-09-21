package commands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/CarlFlo/tally/internal/backup"
	"github.com/CarlFlo/tally/internal/config"
	"github.com/CarlFlo/tally/internal/database"
	restorer "github.com/CarlFlo/tally/internal/restore"
)

const operatorSocketName = "operator.sock"

type operatorRequest struct {
	Action     string `json:"action"`
	ProfileRef string `json:"profile_ref,omitempty"`
	Password   string `json:"password,omitempty"`
	Archive    string `json:"archive,omitempty"`
}

type operatorResponse struct {
	Error string `json:"error,omitempty"`
}

type operatorServer struct {
	listener net.Listener
	path     string
	once     sync.Once
}

func startOperatorServer(ctx context.Context, c config.Config, db *database.Store, backups *backup.Service, events restorer.Publisher, jobs restorer.Scheduler) (*operatorServer, error) {
	path := filepath.Join(c.DataDir, operatorSocketName)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	listener, err := net.Listen("unix", path)
	if err != nil {
		return nil, err
	}
	if err = os.Chmod(path, 0600); err != nil {
		listener.Close()
		_ = os.Remove(path)
		return nil, err
	}
	server := &operatorServer{listener: listener, path: path}
	go server.accept(ctx, c, db, backups, events, jobs)
	go func() {
		<-ctx.Done()
		_ = server.Close()
	}()
	return server, nil
}

func (s *operatorServer) accept(ctx context.Context, c config.Config, db *database.Store, backups *backup.Service, events restorer.Publisher, jobs restorer.Scheduler) {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return
		}
		s.handle(ctx, conn, c, db, backups, events, jobs)
	}
}

func (s *operatorServer) handle(parent context.Context, conn net.Conn, c config.Config, db *database.Store, backups *backup.Service, events restorer.Publisher, jobs restorer.Scheduler) {
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(5 * time.Minute))
	var request operatorRequest
	if err := json.NewDecoder(io.LimitReader(conn, 64<<10)).Decode(&request); err != nil {
		_ = json.NewEncoder(conn).Encode(operatorResponse{Error: "invalid operator request"})
		return
	}
	ctx, cancel := context.WithTimeout(parent, 5*time.Minute)
	defer cancel()
	var err error
	switch request.Action {
	case "reset-password":
		if request.ProfileRef == "" || request.Password == "" {
			err = fmt.Errorf("profile and password are required")
		} else {
			err = resetPasswordValue(ctx, db, c, request.ProfileRef, request.Password)
		}
	case "restore":
		if request.Archive == "" {
			err = fmt.Errorf("archive path is required")
		} else {
			_, err = (restorer.Coordinator{DB: db, Backup: backups, Events: events, Jobs: jobs}).File(ctx, request.Archive)
		}
	default:
		err = fmt.Errorf("unsupported operator action")
	}
	response := operatorResponse{}
	if err != nil {
		response.Error = err.Error()
	}
	_ = json.NewEncoder(conn).Encode(response)
}

func (s *operatorServer) Close() error {
	var closeErr error
	s.once.Do(func() {
		closeErr = s.listener.Close()
		_ = os.Remove(s.path)
	})
	return closeErr
}

func callRunningOperator(ctx context.Context, c config.Config, request operatorRequest) (bool, error) {
	path := filepath.Join(c.DataDir, operatorSocketName)
	var dialer net.Dialer
	conn, err := dialer.DialContext(ctx, "unix", path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) || errors.Is(err, syscall.ENOENT) || errors.Is(err, syscall.ECONNREFUSED) {
			return false, nil
		}
		return false, err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(5 * time.Minute))
	if err = json.NewEncoder(conn).Encode(request); err != nil {
		return true, err
	}
	var response operatorResponse
	if err = json.NewDecoder(io.LimitReader(conn, 64<<10)).Decode(&response); err != nil {
		return true, err
	}
	if response.Error != "" {
		return true, errors.New(response.Error)
	}
	return true, nil
}
