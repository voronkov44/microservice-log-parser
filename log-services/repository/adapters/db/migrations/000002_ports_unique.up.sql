CREATE UNIQUE INDEX IF NOT EXISTS idx_ports_log_node_port_unique
    ON ports(log_id, node_id, port_num);
