CREATE TABLE IF NOT EXISTS logs (
    id              BIGSERIAL PRIMARY KEY,
    file_path       TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'processing',
    nodes_count     INT NOT NULL DEFAULT 0,
    ports_count     INT NOT NULL DEFAULT 0,
    error_text      TEXT NOT NULL DEFAULT '',
    uploaded_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    parsed_at       TIMESTAMPTZ,

    CONSTRAINT logs_status_check CHECK (status IN ('processing', 'parsed', 'failed'))
);

CREATE TABLE IF NOT EXISTS nodes (
    id                  BIGSERIAL PRIMARY KEY,
    log_id              BIGINT NOT NULL REFERENCES logs(id) ON DELETE CASCADE,

    node_guid           TEXT NOT NULL,
    node_desc           TEXT NOT NULL DEFAULT '',
    node_type           INT NOT NULL DEFAULT 0,
    node_kind           TEXT NOT NULL DEFAULT '',
    num_ports           INT NOT NULL DEFAULT 0,

    class_version       INT NOT NULL DEFAULT 0,
    base_version        INT NOT NULL DEFAULT 0,
    system_image_guid   TEXT NOT NULL DEFAULT '',
    port_guid           TEXT NOT NULL DEFAULT '',

    raw_json            TEXT NOT NULL DEFAULT '',

    CONSTRAINT nodes_log_guid_unique UNIQUE (log_id, node_guid)
);

CREATE TABLE IF NOT EXISTS nodes_info (
    id              BIGSERIAL PRIMARY KEY,
    node_id         BIGINT NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,

    node_guid       TEXT NOT NULL,
    serial_number   TEXT NOT NULL DEFAULT '',
    part_number     TEXT NOT NULL DEFAULT '',
    revision        TEXT NOT NULL DEFAULT '',
    product_name    TEXT NOT NULL DEFAULT '',

    raw_json        TEXT NOT NULL DEFAULT '',

    CONSTRAINT nodes_info_node_unique UNIQUE (node_id)
);

CREATE TABLE IF NOT EXISTS ports (
    id                  BIGSERIAL PRIMARY KEY,
    log_id              BIGINT NOT NULL REFERENCES logs(id) ON DELETE CASCADE,
    node_id             BIGINT NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,

    node_guid           TEXT NOT NULL,
    port_guid           TEXT NOT NULL DEFAULT '',
    port_num            INT NOT NULL DEFAULT 0,

    lid                 INT NOT NULL DEFAULT 0,
    local_port_num      INT NOT NULL DEFAULT 0,
    port_state          INT NOT NULL DEFAULT 0,
    port_phy_state      INT NOT NULL DEFAULT 0,
    link_width_active   INT NOT NULL DEFAULT 0,
    link_speed_active   INT NOT NULL DEFAULT 0,

    raw_json            TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_nodes_log_id ON nodes(log_id);
CREATE INDEX IF NOT EXISTS idx_nodes_node_guid ON nodes(node_guid);

CREATE INDEX IF NOT EXISTS idx_nodes_info_node_id ON nodes_info(node_id);
CREATE INDEX IF NOT EXISTS idx_nodes_info_node_guid ON nodes_info(node_guid);

CREATE INDEX IF NOT EXISTS idx_ports_log_id ON ports(log_id);
CREATE INDEX IF NOT EXISTS idx_ports_node_id ON ports(node_id);
CREATE INDEX IF NOT EXISTS idx_ports_node_guid ON ports(node_guid);
