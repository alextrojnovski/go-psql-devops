CREATE TABLE IF NOT EXISTS requests_log (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_created_at ON requests_log(created_at);
