package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"

	"github.com/voronkov44/microservice-log-parser/log-services/repository/core"
)

type DB struct {
	log  *slog.Logger
	conn *sqlx.DB
}

func New(log *slog.Logger, address string) (*DB, error) {
	conn, err := sqlx.Connect("pgx", address)
	if err != nil {
		log.Error("connection problem", "address", address, "error", err)
		return nil, fmt.Errorf("connect db: %w", err)
	}

	return &DB{
		log:  log,
		conn: conn,
	}, nil
}

func (db *DB) Ping(ctx context.Context) error {
	if err := db.conn.PingContext(ctx); err != nil {
		db.log.Warn("db ping failed", "error", err)
		return fmt.Errorf("ping db: %w", err)
	}

	return nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}

func (db *DB) CreateLog(ctx context.Context, filePath string) (int64, error) {
	var id int64

	err := db.conn.QueryRowContext(ctx, `
		INSERT INTO logs (file_path, status)
		VALUES ($1, 'processing')
		RETURNING id
	`, filePath).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("create log: %w", err)
	}

	return id, nil
}

func (db *DB) SaveParsedLog(ctx context.Context, logID int64, parsed core.ParsedLog) (core.SaveParsedLogResult, error) {
	tx, err := db.conn.BeginTxx(ctx, nil)
	if err != nil {
		return core.SaveParsedLogResult{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	var existingLogID int64
	if err := tx.GetContext(ctx, &existingLogID, `
		SELECT id
		FROM logs
		WHERE id = $1
	`, logID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return core.SaveParsedLogResult{}, core.ErrNotFound
		}

		return core.SaveParsedLogResult{}, fmt.Errorf("check log exists: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		DELETE FROM nodes
		WHERE log_id = $1
	`, logID)
	if err != nil {
		return core.SaveParsedLogResult{}, fmt.Errorf("clear old parsed log data: %w", err)
	}

	nodeIDs := make(map[string]int64, len(parsed.Nodes))

	for _, node := range parsed.Nodes {
		if node.NodeGUID == "" {
			return core.SaveParsedLogResult{}, fmt.Errorf("%w: node_guid is empty", core.ErrBadArguments)
		}

		var nodeID int64
		err := tx.QueryRowContext(ctx, `
			INSERT INTO nodes (
				log_id,
				node_guid,
				node_desc,
				node_type,
				node_kind,
				num_ports,
				class_version,
				base_version,
				system_image_guid,
				port_guid,
				raw_json
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
			RETURNING id
		`,
			logID,
			node.NodeGUID,
			node.NodeDesc,
			node.NodeType,
			node.NodeKind,
			node.NumPorts,
			node.ClassVersion,
			node.BaseVersion,
			node.SystemImageGUID,
			node.PortGUID,
			node.RawJSON,
		).Scan(&nodeID)
		if err != nil {
			return core.SaveParsedLogResult{}, fmt.Errorf("insert node %q: %w", node.NodeGUID, err)
		}

		nodeIDs[node.NodeGUID] = nodeID
	}

	for _, info := range parsed.NodesInfo {
		if info.NodeGUID == "" {
			return core.SaveParsedLogResult{}, fmt.Errorf("%w: node_info node_guid is empty", core.ErrBadArguments)
		}

		nodeID, ok := nodeIDs[info.NodeGUID]
		if !ok {
			return core.SaveParsedLogResult{}, fmt.Errorf("%w: node_info references unknown node_guid %q", core.ErrBadArguments, info.NodeGUID)
		}

		_, err := tx.ExecContext(ctx, `
			INSERT INTO nodes_info (
				node_id,
				node_guid,
				serial_number,
				part_number,
				revision,
				product_name,
				raw_json
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`,
			nodeID,
			info.NodeGUID,
			info.SerialNumber,
			info.PartNumber,
			info.Revision,
			info.ProductName,
			info.RawJSON,
		)
		if err != nil {
			return core.SaveParsedLogResult{}, fmt.Errorf("insert node info %q: %w", info.NodeGUID, err)
		}
	}

	for _, port := range parsed.Ports {
		if port.NodeGUID == "" {
			return core.SaveParsedLogResult{}, fmt.Errorf("%w: port node_guid is empty", core.ErrBadArguments)
		}

		nodeID, ok := nodeIDs[port.NodeGUID]
		if !ok {
			return core.SaveParsedLogResult{}, fmt.Errorf("%w: port references unknown node_guid %q", core.ErrBadArguments, port.NodeGUID)
		}

		_, err := tx.ExecContext(ctx, `
			INSERT INTO ports (
				log_id,
				node_id,
				node_guid,
				port_guid,
				port_num,
				lid,
				local_port_num,
				port_state,
				port_phy_state,
				link_width_active,
				link_speed_active,
				raw_json
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		`,
			logID,
			nodeID,
			port.NodeGUID,
			port.PortGUID,
			port.PortNum,
			port.LID,
			port.LocalPortNum,
			port.PortState,
			port.PortPhyState,
			port.LinkWidthActive,
			port.LinkSpeedActive,
			port.RawJSON,
		)
		if err != nil {
			return core.SaveParsedLogResult{}, fmt.Errorf("insert port node_guid=%q port_guid=%q: %w", port.NodeGUID, port.PortGUID, err)
		}
	}

	nodesCount := int32(len(parsed.Nodes))
	portsCount := int32(len(parsed.Ports))

	result, err := tx.ExecContext(ctx, `
		UPDATE logs
		SET
			status = 'parsed',
			nodes_count = $2,
			ports_count = $3,
			error_text = '',
			parsed_at = now()
		WHERE id = $1
	`, logID, nodesCount, portsCount)
	if err != nil {
		return core.SaveParsedLogResult{}, fmt.Errorf("update parsed log status: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return core.SaveParsedLogResult{}, fmt.Errorf("check parsed log update result: %w", err)
	}
	if affected == 0 {
		return core.SaveParsedLogResult{}, core.ErrNotFound
	}

	if err := tx.Commit(); err != nil {
		return core.SaveParsedLogResult{}, fmt.Errorf("commit parsed log: %w", err)
	}

	return core.SaveParsedLogResult{
		LogID:      logID,
		NodesCount: nodesCount,
		PortsCount: portsCount,
	}, nil
}

func (db *DB) FailLog(ctx context.Context, logID int64, errorText string) error {
	result, err := db.conn.ExecContext(ctx, `
		UPDATE logs
		SET
			status = 'failed',
			error_text = $2,
			parsed_at = now()
		WHERE id = $1
	`, logID, errorText)
	if err != nil {
		return fmt.Errorf("fail log: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check fail log result: %w", err)
	}
	if affected == 0 {
		return core.ErrNotFound
	}

	return nil
}

func (db *DB) GetLog(ctx context.Context, logID int64) (core.Log, error) {
	var row logRow

	err := db.conn.GetContext(ctx, &row, `
		SELECT
			id,
			file_path,
			status,
			nodes_count,
			ports_count,
			error_text,
			uploaded_at,
			parsed_at
		FROM logs
		WHERE id = $1
	`, logID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return core.Log{}, core.ErrNotFound
		}

		return core.Log{}, fmt.Errorf("get log: %w", err)
	}

	return row.toCore(), nil
}

func (db *DB) GetNode(ctx context.Context, nodeID int64) (core.Node, error) {
	var row nodeRow

	err := db.conn.GetContext(ctx, &row, `
		SELECT
			id,
			log_id,
			node_guid,
			node_desc,
			node_type,
			node_kind,
			num_ports,
			class_version,
			base_version,
			system_image_guid,
			port_guid,
			raw_json
		FROM nodes
		WHERE id = $1
	`, nodeID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return core.Node{}, core.ErrNotFound
		}

		return core.Node{}, fmt.Errorf("get node: %w", err)
	}

	node := row.toCore()

	info, err := db.getNodeInfoByNodeID(ctx, nodeID)
	if err != nil {
		return core.Node{}, err
	}
	node.Info = info

	return node, nil
}

func (db *DB) GetPortsByNode(ctx context.Context, nodeID int64) ([]core.Port, error) {
	var rows []portRow

	err := db.conn.SelectContext(ctx, &rows, `
		SELECT
			id,
			log_id,
			node_id,
			node_guid,
			port_guid,
			port_num,
			lid,
			local_port_num,
			port_state,
			port_phy_state,
			link_width_active,
			link_speed_active,
			raw_json
		FROM ports
		WHERE node_id = $1
		ORDER BY port_num, id
	`, nodeID)
	if err != nil {
		return nil, fmt.Errorf("get ports by node: %w", err)
	}

	out := make([]core.Port, 0, len(rows))
	for _, row := range rows {
		out = append(out, row.toCore())
	}

	return out, nil
}

func (db *DB) GetNodesByLog(ctx context.Context, logID int64) ([]core.Node, error) {
	var rows []nodeRow

	err := db.conn.SelectContext(ctx, &rows, `
		SELECT
			id,
			log_id,
			node_guid,
			node_desc,
			node_type,
			node_kind,
			num_ports,
			class_version,
			base_version,
			system_image_guid,
			port_guid,
			raw_json
		FROM nodes
		WHERE log_id = $1
		ORDER BY id
	`, logID)
	if err != nil {
		return nil, fmt.Errorf("get nodes by log: %w", err)
	}

	out := make([]core.Node, 0, len(rows))
	for _, row := range rows {
		node := row.toCore()

		info, err := db.getNodeInfoByNodeID(ctx, node.ID)
		if err != nil {
			return nil, err
		}
		node.Info = info

		out = append(out, node)
	}

	return out, nil
}

func (db *DB) GetPortsByLog(ctx context.Context, logID int64) ([]core.Port, error) {
	var rows []portRow

	err := db.conn.SelectContext(ctx, &rows, `
		SELECT
			id,
			log_id,
			node_id,
			node_guid,
			port_guid,
			port_num,
			lid,
			local_port_num,
			port_state,
			port_phy_state,
			link_width_active,
			link_speed_active,
			raw_json
		FROM ports
		WHERE log_id = $1
		ORDER BY node_id, port_num, id
	`, logID)
	if err != nil {
		return nil, fmt.Errorf("get ports by log: %w", err)
	}

	out := make([]core.Port, 0, len(rows))
	for _, row := range rows {
		out = append(out, row.toCore())
	}

	return out, nil
}

func (db *DB) getNodeInfoByNodeID(ctx context.Context, nodeID int64) (*core.NodeInfo, error) {
	var row nodeInfoRow

	err := db.conn.GetContext(ctx, &row, `
		SELECT
			id,
			node_id,
			node_guid,
			serial_number,
			part_number,
			revision,
			product_name,
			raw_json
		FROM nodes_info
		WHERE node_id = $1
	`, nodeID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}

		return nil, fmt.Errorf("get node info by node id: %w", err)
	}

	info := row.toCore()
	return &info, nil
}

type logRow struct {
	ID         int64        `db:"id"`
	FilePath   string       `db:"file_path"`
	Status     string       `db:"status"`
	NodesCount int32        `db:"nodes_count"`
	PortsCount int32        `db:"ports_count"`
	Error      string       `db:"error_text"`
	UploadedAt time.Time    `db:"uploaded_at"`
	ParsedAt   sql.NullTime `db:"parsed_at"`
}

func (r logRow) toCore() core.Log {
	var parsedAt string
	if r.ParsedAt.Valid {
		parsedAt = formatTimestamp(r.ParsedAt.Time)
	}

	return core.Log{
		ID:         r.ID,
		FilePath:   r.FilePath,
		Status:     core.LogStatus(r.Status),
		NodesCount: r.NodesCount,
		PortsCount: r.PortsCount,
		Error:      r.Error,
		UploadedAt: formatTimestamp(r.UploadedAt),
		ParsedAt:   parsedAt,
	}
}

func formatTimestamp(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

type nodeRow struct {
	ID              int64  `db:"id"`
	LogID           int64  `db:"log_id"`
	NodeGUID        string `db:"node_guid"`
	NodeDesc        string `db:"node_desc"`
	NodeType        int32  `db:"node_type"`
	NodeKind        string `db:"node_kind"`
	NumPorts        int32  `db:"num_ports"`
	ClassVersion    int32  `db:"class_version"`
	BaseVersion     int32  `db:"base_version"`
	SystemImageGUID string `db:"system_image_guid"`
	PortGUID        string `db:"port_guid"`
	RawJSON         string `db:"raw_json"`
}

func (r nodeRow) toCore() core.Node {
	return core.Node{
		ID:              r.ID,
		LogID:           r.LogID,
		NodeGUID:        r.NodeGUID,
		NodeDesc:        r.NodeDesc,
		NodeType:        r.NodeType,
		NodeKind:        r.NodeKind,
		NumPorts:        r.NumPorts,
		ClassVersion:    r.ClassVersion,
		BaseVersion:     r.BaseVersion,
		SystemImageGUID: r.SystemImageGUID,
		PortGUID:        r.PortGUID,
		RawJSON:         r.RawJSON,
	}
}

type nodeInfoRow struct {
	ID           int64  `db:"id"`
	NodeID       int64  `db:"node_id"`
	NodeGUID     string `db:"node_guid"`
	SerialNumber string `db:"serial_number"`
	PartNumber   string `db:"part_number"`
	Revision     string `db:"revision"`
	ProductName  string `db:"product_name"`
	RawJSON      string `db:"raw_json"`
}

func (r nodeInfoRow) toCore() core.NodeInfo {
	return core.NodeInfo{
		ID:           r.ID,
		NodeID:       r.NodeID,
		NodeGUID:     r.NodeGUID,
		SerialNumber: r.SerialNumber,
		PartNumber:   r.PartNumber,
		Revision:     r.Revision,
		ProductName:  r.ProductName,
		RawJSON:      r.RawJSON,
	}
}

type portRow struct {
	ID              int64  `db:"id"`
	LogID           int64  `db:"log_id"`
	NodeID          int64  `db:"node_id"`
	NodeGUID        string `db:"node_guid"`
	PortGUID        string `db:"port_guid"`
	PortNum         int32  `db:"port_num"`
	LID             int32  `db:"lid"`
	LocalPortNum    int32  `db:"local_port_num"`
	PortState       int32  `db:"port_state"`
	PortPhyState    int32  `db:"port_phy_state"`
	LinkWidthActive int32  `db:"link_width_active"`
	LinkSpeedActive int32  `db:"link_speed_active"`
	RawJSON         string `db:"raw_json"`
}

func (r portRow) toCore() core.Port {
	return core.Port{
		ID:              r.ID,
		LogID:           r.LogID,
		NodeID:          r.NodeID,
		NodeGUID:        r.NodeGUID,
		PortGUID:        r.PortGUID,
		PortNum:         r.PortNum,
		LID:             r.LID,
		LocalPortNum:    r.LocalPortNum,
		PortState:       r.PortState,
		PortPhyState:    r.PortPhyState,
		LinkWidthActive: r.LinkWidthActive,
		LinkSpeedActive: r.LinkSpeedActive,
		RawJSON:         r.RawJSON,
	}
}
