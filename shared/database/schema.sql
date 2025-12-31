-- Messages table with idempotency_id as unique constraint
CREATE TABLE IF NOT EXISTS messages (
    idempotency_id VARCHAR(255) PRIMARY KEY,
    correlation_id VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    payload JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_correlation_id (correlation_id),
    INDEX idx_status (status)
);

-- Message history table to track all status changes
CREATE TABLE IF NOT EXISTS message_history (
    id SERIAL PRIMARY KEY,
    idempotency_id VARCHAR(255) NOT NULL,
    correlation_id VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL,
    service_name VARCHAR(100) NOT NULL,
    event_id VARCHAR(255),
    error_message TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (idempotency_id) REFERENCES messages(idempotency_id) ON DELETE CASCADE,
    INDEX idx_idempotency_id (idempotency_id),
    INDEX idx_correlation_id (correlation_id),
    INDEX idx_created_at (created_at)
);

